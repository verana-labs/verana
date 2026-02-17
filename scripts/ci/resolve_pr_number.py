#!/usr/bin/env python3
import json
import os
import sys
import urllib.request


def main() -> int:
    branch = os.environ.get("BRANCH_REF", "")
    repo = os.environ.get("GITHUB_REPOSITORY", "")
    owner = os.environ.get("GITHUB_REPOSITORY_OWNER", "")
    token = os.environ.get("GITHUB_TOKEN", "")

    if not (branch and repo and owner and token):
        return 0

    url = f"https://api.github.com/repos/{repo}/pulls?head={owner}:{branch}&state=open"
    req = urllib.request.Request(
        url,
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except Exception:
        return 0

    if data:
        pr_number = data[0].get("number")
        if pr_number:
            print(pr_number)
    return 0


if __name__ == "__main__":
    sys.exit(main())
