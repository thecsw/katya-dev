import scrapy
from scrapy.linkextractors import LinkExtractor

from icecream import ic

class KatyaSpider(scrapy.Spider):
    name = "katya"

    # allowed_domains = ["sandyuraz.com"]
    # start_urls = ["https://sandyuraz.com/arts"]
    allowed_domains = ["www.drive2.ru"]
    start_urls = ["https://www.drive2.ru/communities/14/blog/"]

    link_extractor = LinkExtractor
    start_url = start_urls[0]

    def parse(self, response):
        links = self.link_extractor(
            #allow=self.start_url + ".*", # uncomment if you want to match subpaths only
            deny="#",  # don't match sections of the same webpage
            restrict_css="a",
            unique=True,
            allow_domains=self.allowed_domains,
        ).extract_links(response)

        #ic(links)

        yield {
            "name": self.name,
            "start": self.start_url,
            "url": response.url,
            "ip": response.ip_address.exploded,
            "status": response.status,
            "text": response.text,
        }

        # Follow all accompanying links
        yield from response.follow_all(links, self.parse)
