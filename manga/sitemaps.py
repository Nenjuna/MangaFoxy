from django.contrib.sitemaps import Sitemap
from django.urls import reverse
from .models import Manga

class StaticViewSitemap(Sitemap):
    priority = 0.5
    changefreq = 'weekly'

    def items(self):
        return ['home']  # Named URLs

    def location(self, item):
        return reverse(item)

class MangaSitemap(Sitemap):
    changefreq = 'weekly'
    priority = 0.8

    def items(self):
        return Manga.objects.all()

    # def lastmod(self, obj):
    #     return obj.updated_at  # if you have a field for last updated

    def location(self, obj):
        return reverse('manga-detail', args=[obj.slug])
