import json
import requests
from typing import List

from itemadapter import ItemAdapter
from bs4 import BeautifulSoup

# URL to submit processed strings
URL_BASE = "http://127.0.0.1:32000"
URL_CLEAN = URL_BASE + "/text"
URL_STATUS = URL_BASE + "/status"

YAGAMI_URL = "http://127.0.0.1:32393/process"

# Session for requests
s = requests.session()
s.verify = False


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
                ).encode("utf-8"),
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
        # clean_text = clean_raw_html(str(text))
        clean_text = " ".join(extract_text(text))

        # Send the text to yagami for processing. Yagami will submit it later
        try:
            print("SENDING TO YAGAMI", ItemAdapter(item).get("title"))
            s.post(
                YAGAMI_URL,
                data=json.dumps(
                    {
                        "title": ItemAdapter(item).get("title"),
                        "ip": ItemAdapter(item).get("ip"),
                        "url": ItemAdapter(item).get("url"),
                        "start": spider.start_url,
                        "status": ItemAdapter(item).get("status"),
                        "crawler": spider.name,
                        "text": clean_text,
                    },
                    ensure_ascii=False,
                    sort_keys=True,
                ).encode("utf-8"),
                headers=headers(spider.name),
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
