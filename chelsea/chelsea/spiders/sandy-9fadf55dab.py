import scrapy
from scrapy.linkextractors import LinkExtractor


class TemplateSpider(scrapy.Spider):
    name = "sandy-9fadf55dab"

    allowed_domains = ["sandyuraz.com"]
    start_urls = ["https://sandyuraz.com/blogs"]

    link_extractor = LinkExtractor
    start_url = start_urls[0]

    def parse(self, response):
        links = self.link_extractor(
            allow=self.start_url + ".*", # uncomment if you want to match subpaths only
            deny="#",  # don't match sections of the same webpage
            #restrict_css="a",
            restrict_xpaths="//a",
            unique=True,
            allow_domains=self.allowed_domains,
        ).extract_links(response)

        titleMatches = response.xpath("/html/head/title/text()").extract()
        title = "unknown"
        if len(titleMatches) > 0:
            title = titleMatches[0]

        yield {
            "url": response.url,
            "ip": response.ip_address.exploded,
            "status": response.status,
            "text": response.text,
            "title": title,
        }

        # Follow all accompanying links
        yield from response.follow_all(links, self.parse)
