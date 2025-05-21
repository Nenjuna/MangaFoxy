"""
URL configuration for mangabase project.

The `urlpatterns` list routes URLs to views. For more information please see:
    https://docs.djangoproject.com/en/4.2/topics/http/urls/
Examples:
Function views
    1. Add an import:  from my_app import views
    2. Add a URL to urlpatterns:  path('', views.home, name='home')
Class-based views
    1. Add an import:  from other_app.views import Home
    2. Add a URL to urlpatterns:  path('', Home.as_view(), name='home')
Including another URLconf
    1. Import the include() function: from django.urls import include, path
    2. Add a URL to urlpatterns:  path('blog/', include('blog.urls'))
"""
from django.contrib import admin
from django.urls import path, include
from . import views
from django.views.generic import TemplateView
from django.contrib.sitemaps.views import sitemap
from .sitemaps import StaticViewSitemap, GenreSitemap, MangaSitemap, ChapterSitemap

sitemaps = {
    'static': StaticViewSitemap,
    'genre': GenreSitemap,
    'manga': MangaSitemap,
    'chapter': ChapterSitemap,
}

urlpatterns = [
    path('admin/', admin.site.urls),
    path("__reload__/", include("django_browser_reload.urls")),
    path('', views.home, name='home'),
    path('api/', include('manga.urls')),
    path('genre/', views.genre_view, name='genre-list'),
    path('copyright/', views.copyright_view, name='copyright'),
    path('terms/', views.terms_view, name='terms'),
    path('genre/<slug:genre_slug>/', views.genre_view, name='genre-detail'),
    path('about/', views.about_view, name='about'),
    path('contact/', views.contact_view, name='contact'),
    path('updates/', views.updates_view, name='updates'),
    path('<slug:slug>/', views.manga_detail_view, name='manga-detail'),
    path('<slug:manga_slug>/<slug:chapter_number>/',
         views.chapter_detail_view, name='chapter_detail'),
    path('sitemap.xml', views.sitemap_index, name='sitemap_index'),
    path('sitemap-manga.xml', views.sitemap_manga, name='sitemap_manga'),
    path('sitemap-chapters-<int:page>.xml',
         views.sitemap_chapters, name='sitemap_chapters'),
    path('sitemap-genres.xml', views.sitemap_genres, name='sitemap_genres'),
    path('sitemap-pages.xml', views.sitemap_pages, name='sitemap_pages'),
    path("robots.txt", TemplateView.as_view(
        template_name="robots.txt", content_type="text/plain")),
    path("manifest.json", TemplateView.as_view(
        template_name="manifest.json", content_type="application/json"), name='manifest'),
]
