#!/usr/bin/env python3
import sys
import json
import base64
import requests
from requests.auth import HTTPBasicAuth

KATYA_URL = "http://127.0.0.1:32000/source"
YAGAMI_URL = "http://127.0.0.1:32393/process"

s = requests.session()
s.verify = False


def main():
    if len(sys.argv) < 3:
        print("need a user:pass and filename")
        exit(1)

    userpass = sys.argv[1].split(":")
    username = userpass[0]
    password = userpass[1]
    filename = sys.argv[2]
    data = ""
    with open(filename) as f:
        data = f.read()

    print("Creating the source in Katya")
    r = s.post(
        KATYA_URL,
        data=json.dumps(
            {
                "link": filename,
                "label": filename,
                "only_subpaths": True,
            },
            ensure_ascii=False,
            sort_keys=True,
        ).encode("utf-8"),
        auth=HTTPBasicAuth(username, password),
        headers={
            "Content-Type": "application/json",
        },
    )
    print(r)

    print("Sending request to Yagami")
    payload = json.dumps(
        {
            "title": filename,
            "ip": filename,
            "url": filename,
            "start": filename,
            "status": 100,
            "crawler": "LOCAL_UPLOAD",
            "text": data,
        },
        ensure_ascii=False,
        sort_keys=True,
    ).encode("utf-8")

    r = s.post(
        YAGAMI_URL,
        data=payload,
        headers={
            "User-Agent": "LOCAL_UPLOAD",
            "Accept": "*/*",
            "Connection": "keep-alive",
            "Content-Type": "application/json",
            "Authorization": "cool_local_key",
        },
    )
    print(r.text)


print("starting the main")
if __name__ == "__main__":
    main()
