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
from icecream import ic

# URL to submit processed strings
URL_CLEAN = "http://127.0.0.1:10000/noor"
URL_STATUS = "http://127.0.0.1:10000/status"

SESSION = requests.session()


class NoorPipeline:
    def open_spider(self, spider):
        self.file = open(f"spider-{spider.name}.json", "w")
        requests.post(
            URL_STATUS, data=json.dumps({"status": f"started spider {spider.name}"})
        )

    def close_spider(self, spider):
        self.file.close()
        requests.post(
            URL_STATUS, data=json.dumps({"status": f"closed spider {spider.name}"})
        )

    def process_item(self, item, spider):
        text = ItemAdapter(item).get("text")
        clean_text = clean_raw_html(str(text))

        to_return = {
            "text": clean_text,
            "ip": ItemAdapter(item).get("ip"),
            "url": ItemAdapter(item).get("url"),
            "status": ItemAdapter(item).get("status"),
            "start": ItemAdapter(item).get("start"),
            "name": ItemAdapter(item).get("name"),
        }

        final_json = json.dumps(to_return, ensure_ascii=False, sort_keys=True)
        self.file.write(final_json)

        requests.post(URL_CLEAN, data=final_json.encode("utf-8"), headers={})

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
    return str(requests.get(url).content)


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
    text = extract_text(raw_html)
    return clean_text(" ".join(text))
