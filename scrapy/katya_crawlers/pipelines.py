# Define your item pipelines here
#
# Don't forget to add your pipeline to the ITEM_PIPELINES setting
# See: https://docs.scrapy.org/en/latest/topics/item-pipeline.html

# useful for handling different item types with a single interface
import json
import requests
from typing import List

from itemadapter import ItemAdapter
from bs4 import BeautifulSoup
from cleantext import clean

# from icecream import ic
# import langdetect
# import nltk
import spacy

# URL to submit processed strings
URL_BASE = "http://127.0.0.1:32000"
URL_CLEAN = URL_BASE + "/text"
URL_STATUS = URL_BASE + "/status"

# Session for requests
s = requests.session()
s.verify = False

# ---------- RUSSIAN SPACY -------------
# Check here: https://spacy.io/models/ru
# --------------------------------------
nlp_ru = spacy.load('ru_core_news_sm') # lightweight
#nlp_ru = spacy.load("ru_core_news_lg")  # heavylifter


def headers(spiderName):
    return {
        "User-Agent": spiderName,
        "Accept": "*/*",
        "Connection": "keep-alive",
        "Content-Type": "application/json",
        "Authorization": "cool_local_key",
    }


class ScrapyPipeline:
    def open_spider(self, spider):
        """
        This runs when a new spider starts running. We send this
        to katya, so that we can add a new run.
        """
        # self.file = open(f"spider-{spider.name}.json", "w")
        try:
            s.post(
                URL_STATUS,
                data=json.dumps(
                    {
                        "status": "started",
                        "name": spider.name,
                    }
                ),
                headers=headers(spider.name),
            )
        except Exception as e:
            print(f"Failed to send started status of {spider.name}:", e)

    def close_spider(self, spider):
        """
        This runs when a spider finishes its job. We send the status
        to katya to update the Runs table.
        """
        # self.file.close()
        try:
            s.post(
                URL_STATUS,
                data=json.dumps(
                    {
                        "status": "finished",
                        "name": spider.name,
                    }
                ),
                headers=headers(spider.name),
            )
        except Exception as e:
            print(f"Failed to send finished status of {spider.name}:", e)

    def process_item(self, item, spider):
        """
        This function is called when a crawler yields results, which is the
        payload that a crawler sends back to us. We clean the received payload,
        which is a raw HTML read of the page and send it in a text payload straight
        to katya.

        TODO: Possibly in the future, it would need to do some automated lexical tagging.
        """
        text = ItemAdapter(item).get("text")
        clean_text = clean_raw_html(str(text))

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
            "title": ItemAdapter(item).get("title"),
            "ip": ItemAdapter(item).get("ip"),
            "url": ItemAdapter(item).get("url"),
            "status": ItemAdapter(item).get("status"),
            "start": spider.start_url,
            "name": spider.name,
            "num_words": num_words,
            "num_sentences": num_sentences,
        }

        final_json = json.dumps(to_return, ensure_ascii=False, sort_keys=True)
        # self.file.write(final_json)

        try:
            s.post(
                URL_CLEAN, data=final_json.encode("utf-8"), headers=headers(spider.name)
            )
        except Exception as e:
            print("Failed to send a text payload:", e)

        return item


SOUP_PARSER = "html5lib"  # or "lxml"

BAD_SOUP_TAGS = [
    "[document]",
    "noscript",
    "header",
    "html",
    "meta",
    "head",
    "input",
    "script",
    "style",
    "title",
]

TO_REMOVE = [
    "старая версия для apache:",
    "новая версия для nginx: end",
    "новая версия для nginx: begin",
    "goto old version message",
    "Yandex.Metrika counter",
    "</h3>",
    "<h3>",
    "<p>",
    "</p>",
    "<strong>",
    "</strong>",
    "=  !=",
    "/Logo",
    "if expr",
    "||",
    "&&",
    "!=" '="=',
    '="',
    "''",
    "/noindex",
    "noindex",
]


def request_html(url: str) -> str:
    """
    Takes a URL in a string format and returns the HTML
    page in bytes format.
    """
    return str(s.get(url).content)


def extract_text(html_page: str) -> List[str]:
    """
    Takes an HTML page in bytes format and returns "clean"
    text from the HTML, as in excluding elements that are in
    the blacklist.
    """
    # Create a soup instance
    soup = BeautifulSoup(html_page, features=SOUP_PARSER)
    # Extract all tags and elements
    all_elements = soup.find_all(text=True)

    # Remove tags and elements that are just titles, styles, links, etc.
    to_return: List[str] = []
    for element in all_elements:
        if element.parent == None or element.parent.name in BAD_SOUP_TAGS:
            continue
        to_return.append(str(element))

    return to_return


def clean_text(dirty_text: str) -> str:
    """
    Cleans up the text from annoying newlines, tabs, whitespaces, and
    extra glyphs.
    """
    # Some hardcoded strings to remove
    for to_remove in TO_REMOVE:
        dirty_text = dirty_text.replace(to_remove, "")
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


def clean_raw_html(raw_html: str) -> str:
    """
    Automatically cleans a raw html file to extract pure text.
    """
    text = extract_text(raw_html)
    return clean_text(" ".join(text))
