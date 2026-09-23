import unittest
from promote import release


class ReleaseTest(unittest.TestCase):
    def image(self, name):
        return {"repository": f"123456789012.dkr.ecr.us-west-2.amazonaws.com/kubevista-{name}", "digest": "sha256:" + "a" * 64, "revision": "b" * 40}

    def test_preserves_digests(self):
        result = release(self.image("api"), self.image("web"))
        self.assertEqual(result["image"]["digest"], "sha256:" + "a" * 64)
        self.assertEqual(result["webImage"]["tag"], "")

    def test_rejects_mixed_commits(self):
        with self.assertRaises(ValueError):
            release(self.image("api"), {**self.image("web"), "revision": "c" * 40})

    def test_rejects_mutable_tags(self):
        with self.assertRaises(ValueError):
            release({**self.image("api"), "digest": "latest"}, self.image("web"))

    def test_rejects_wrong_repository(self):
        with self.assertRaises(ValueError):
            release(self.image("web"), self.image("web"))
