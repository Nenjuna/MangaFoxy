from django.shortcuts import render, get_object_or_404
from manga.models import Manga
import requests
from bs4 import BeautifulSoup

def scrape_chapter(slug):
    header = {'User-agent': 'Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36'}
    url = "https://mangaowl.io/read-1/" + slug + "/ajax/chapters/"
    chapter_links = requests.post(url, headers=header)
    lik_html = BeautifulSoup(chapter_links.text, 'lxml')
    # print(lik_html)
    chap = lik_html.find("ul")
    chap_links = [{c['href']: c.text.strip()} for c in chap.find_all("a")]
    # print(slug)
    # print(chap_links)
    seen = set()
    cleaned = []

    for entry in chap_links:
        for url, title in entry.items():
            if title and url not in seen:
                cleaned.append({url: title})
                seen.add(url)
    return cleaned

def home(request):
    mangas = Manga.objects.all()[:20]
    return render(request, 'index.html', {'mangas': mangas})

def manga_detail_view(request, slug):
    manga = get_object_or_404(Manga, slug=slug)
    chapters = scrape_chapter(slug)
    return render(request, 'manga_detail.html', {'manga': manga, 'chapters': chapters})
