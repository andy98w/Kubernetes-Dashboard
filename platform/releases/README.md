# Release digests

The publisher creates `staging.json` in a review PR after both images pass CI,
vulnerability scanning and signing. Production promotion copies those exact
digests into `production.json`; it never builds another image.

No initial digest files are provided: old demo images must not look like a newly
validated release. Argo Applications below intentionally fail to render until
their release file exists. Install them only after provisioning and review.
