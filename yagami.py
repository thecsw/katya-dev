#!/usr/bin/env python3
print("[YAGAMI] importing libraries")
import sys
import os
import json
import requests
from typing import List
from cleantext import clean
import spacy
import threading
import queue
from flask import Flask, request, json

print("[YAGAMI] defining constants")
# URL to submit processed strings
URL_BASE = "http://127.0.0.1:32000"
URL_CLEAN = URL_BASE + "/text"
URL_STATUS = URL_BASE + "/status"

YAGAMI_HOST = "localhost"
YAGAMI_PORT = 32393

# Session for requests
s = requests.session()
s.verify = False

# This is for worker to work through
workerQueue = queue.Queue()

print("[YAGAMI] loading dictionaries")
# ---------- RUSSIAN SPACY -------------
# Check here: https://spacy.io/models/ru
# --------------------------------------
# nlp_ru = spacy.load("ru_core_news_sm")  # lightweight
nlp_ru = spacy.load("ru_core_news_lg")  # heavylifter

app = Flask(__name__)


@app.route("/", methods=["GET"])
def hello():
    name = request.args.get("name", "World")
    return f"[YAGAMI] Hello, {name}!"


@app.route("/process", methods=["POST"])
def process():
    if not request.is_json:
        return "Expected a json with a text key"
    data = request.get_json(force=True)
    if data is None:
        return "JSON parsing failed"
    print(f"[YAGAMI] Put a new worker job: {data['title']}")
    workerQueue.put(data)
    return "success"


CHUNK_SIZE = 100_000

def fragmentize(text: str) -> List[str]:
    to_return = []
    for i in range(0, len(text), CHUNK_SIZE):
        to_return.append(text[i : i + CHUNK_SIZE])
    return to_return


def worker():
    while True:
        data = workerQueue.get()
        fragments = fragmentize(data["text"])
        print(f"[YAGAMI] {data['url']} got fragmented into {len(fragments)} pieces")
        for i, fragment in enumerate(fragments):
            print(f"[YAGAMI] Sending {data['title']} with count = {i}")
            p_analyze(
                data["ip"],
                data["url"],
                int(data["status"]),
                data["start"],
                i,
                data["crawler"],
                data["title"],
                fragment,
            )
        print(f"[YAGAMI] Worker completed {data['title']}")


def p_analyze(
    ip: str,
    url: str,
    status: int,
    start: str,
    count: int,
    crawler: str,
    title: str,
    text: str,
):
    clean_text = clean_text_f(text)
    doc = nlp_ru(clean_text)

    # word_tokens = nltk.word_tokenize(clean_text, language="russian")
    # num_words = len(word_tokens)

    # sent_tokens = nltk.sent_tokenize(clean_text, language="russian")
    # num_sentences = len(sent_tokens)

    num_sentences = len([sent for sent in doc.sents])
    num_words = len([True for token in doc if token.is_alpha])

    shapes = " ".join(([token.shape_ for token in doc]))
    tags = " ".join(([token.tag_ for token in doc]))
    lemmas = " ".join(([token.lemma_ for token in doc]))
    to_send_text = " ".join(([token.text for token in doc]))

    to_return = {
        "original": clean_text,
        "text": to_send_text,
        "shapes": shapes,
        "tags": tags,
        "lemmas": lemmas,
        "title": title,
        "ip": ip,
        "url": f"{url}#{count}",
        "status": status,
        "start": f"{start}",
        "name": crawler,
        "num_words": num_words,
        "num_sentences": num_sentences,
    }

    final_json = json.dumps(to_return, ensure_ascii=False, sort_keys=True)
    # self.file.write(final_json)

    try:
        s.post(
            URL_CLEAN,
            data=final_json.encode("utf-8"),
            headers=getSpiderHeaders("LOCAL"),
        )
    except Exception as e:
        print("Failed to send a text payload:", e)


def getSpiderHeaders(spiderName):
    return {
        "User-Agent": spiderName,
        "Accept": "*/*",
        "Connection": "keep-alive",
        "Content-Type": "application/json",
        "Authorization": "cool_local_key",
    }


def clean_text_f(dirty_text: str) -> str:
    """
    Cleans up the text from annoying newlines, tabs, whitespaces, and
    extra glyphs.
    """
    # Use the text cleaner to remove scary stuff
    return clean(
        dirty_text,
        fix_unicode=True,
        to_ascii=False,
        lower=False,
        no_line_breaks=True,
        no_urls=True,
        no_emails=True,
        no_phone_numbers=True,
        no_numbers=False,
        no_digits=False,
        no_currency_symbols=False,
        no_punct=False,
        no_emoji=False,
        replace_with_punct="",
        replace_with_url="<URL>",
        replace_with_email="<EMAIL>",
        replace_with_phone_number="<PHONE>",
        replace_with_number="<NUMBER>",
        replace_with_digit="0",
        replace_with_currency_symbol="<CUR>",
        lang="en",
    )


if __name__ == "__main__":
    wrk = threading.Thread(target=worker, args=(), name="Scapy worker")
    wrk.start()
    print("[YAGAMI] Started the spacy worker")
    app.run(host=YAGAMI_HOST, port=YAGAMI_PORT, debug=False)
