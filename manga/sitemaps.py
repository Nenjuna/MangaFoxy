from django.contrib.sitemaps import Sitemap
from django.urls import reverse
from manga.models import Manga, Chapter
from django.utils import timezone


class StaticViewSitemap(Sitemap):
    priority = 1.0
    changefreq = 'daily'

    def items(self):
        return ['home', 'copyright', 'terms']

    def location(self, item):
        if item == 'home':
            return reverse('home')
        return reverse(item)


class GenreSitemap(Sitemap):
    priority = 0.8
    changefreq = 'daily'

    def items(self):
        # Get all unique genres
        genres = Manga.objects.values_list('genre', flat=True).distinct()
        unique_genres = set()
        for genre_list in genres:
            if genre_list:
                unique_genres.update(genre_list)
        return sorted(list(unique_genres))

    def location(self, item):
        return f'/genre/{item}/'


class MangaSitemap(Sitemap):
    priority = 0.9
    changefreq = 'daily'

    def items(self):
        return Manga.objects.all()

    def lastmod(self, obj):
        return obj.updated_at or timezone.now()


class ChapterSitemap(Sitemap):
    priority = 0.7
    changefreq = 'weekly'

    def items(self):
        return Chapter.objects.all()

    def lastmod(self, obj):
        return obj.updated_at or timezone.now()

    def location(self, obj):
        return f'/{obj.manga.slug}/{obj.slug}/'
