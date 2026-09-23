"""Create Helm JSON values from verified publisher artifacts; never rebuild."""
import argparse
import json
import re
from pathlib import Path


def validate(image, name):
    if not re.fullmatch(r"\d{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com/" + name, image.get("repository", "")):
        raise ValueError("Unexpected ECR repository")
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", image.get("digest", "")):
        raise ValueError("A full immutable SHA256 digest is required")
    return {"repository": image["repository"], "digest": image["digest"], "tag": ""}


def release(api, web):
    if api.get("revision") != web.get("revision") or not re.fullmatch(r"[0-9a-f]{40}", api.get("revision", "")):
        raise ValueError("Images must come from the same commit")
    return {"image": validate(api, "kubevista-api"), "webImage": validate(web, "kubevista-web"), "releaseRevision": api["revision"]}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("environment", choices=["staging", "production"])
    parser.add_argument("--artifacts", type=Path, default=Path("release"))
    args = parser.parse_args()
    root = Path("platform/releases")
    if args.environment == "staging":
        data = release(*(json.loads((args.artifacts / f"kubevista-{name}.json").read_text()) for name in ("api", "web")))
    else:
        staging = json.loads((root / "staging.json").read_text())
        data = release({**staging["image"], "revision": staging["releaseRevision"]}, {**staging["webImage"], "revision": staging["releaseRevision"]})
    root.mkdir(parents=True, exist_ok=True)
    (root / f"{args.environment}.json").write_text(json.dumps(data, indent=2) + "\n")


if __name__ == "__main__":
    main()
