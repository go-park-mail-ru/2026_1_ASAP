#!/usr/bin/env python3

import argparse
import base64
import json
import sys


def target_headers(auth_cookie: str, csrf_token: str, content_type: bool) -> dict[str, list[str]]:
    headers = {
        "Cookie": [auth_cookie],
        "X-CSRF-TOKEN": [csrf_token],
    }
    if content_type:
        headers["Content-Type"] = ["application/json"]
    return headers


def write_create_targets(args: argparse.Namespace) -> None:
    url = args.base_url.rstrip("/") + "/api/v1/chats/"
    headers = target_headers(args.auth_cookie, args.csrf_token, content_type=True)

    with open(args.output, "w", encoding="utf-8") if args.output != "-" else sys.stdout as out:
        for idx in range(1, args.count + 1):
            body = json.dumps(
                {
                    "type": "group",
                    "title": f"{args.title_prefix}-{idx:06d}",
                    "members_id": [],
                },
                ensure_ascii=False,
                separators=(",", ":"),
            ).encode("utf-8")
            target = {
                "method": "POST",
                "url": url,
                "header": headers,
                "body": base64.b64encode(body).decode("ascii"),
            }
            out.write(json.dumps(target, separators=(",", ":")) + "\n")


def write_read_targets(args: argparse.Namespace) -> None:
    url = args.base_url.rstrip("/") + "/api/v1/chats/"
    target = {
        "method": "GET",
        "url": url,
        "header": target_headers(args.auth_cookie, args.csrf_token, content_type=False),
    }

    with open(args.output, "w", encoding="utf-8") if args.output != "-" else sys.stdout as out:
        out.write(json.dumps(target, separators=(",", ":")) + "\n")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate vegeta JSON targets.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--base-url", required=True)
    common.add_argument("--auth-cookie", required=True)
    common.add_argument("--csrf-token", required=True)
    common.add_argument("--output", required=True)

    create = subparsers.add_parser("create", parents=[common])
    create.add_argument("--count", type=int, required=True)
    create.add_argument("--title-prefix", default="perf-group")
    create.set_defaults(func=write_create_targets)

    read = subparsers.add_parser("read", parents=[common])
    read.set_defaults(func=write_read_targets)

    args = parser.parse_args()
    args.func(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

