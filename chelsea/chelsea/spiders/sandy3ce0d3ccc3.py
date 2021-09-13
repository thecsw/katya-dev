import scrapy
from scrapy.linkextractors import LinkExtractor

class TemplateSpider(scrapy.Spider):
    name = "sandy3ce0d3ccc3"

    allowed_domains = ["sandyuraz.com"]
    start_urls = ["https://sandyuraz.com"]
    
    link_extractor = LinkExtractor
    start_url = start_urls[0]

    def parse(self, response):
        links = self.link_extractor(
            allow=self.start_url + ".*", # uncomment if you want to match subpaths only
            deny="#",  # don't match sections of the same webpage
            restrict_css="a",
            unique=True,
            allow_domains=self.allowed_domains,
        ).extract_links(response)

        yield {
            "url": response.url,
            "ip": response.ip_address.exploded,
            "status": response.status,
            "text": response.text,
        }

        # Follow all accompanying links
        yield from response.follow_all(links, self.parse)
