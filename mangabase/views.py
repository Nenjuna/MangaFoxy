from django.shortcuts import render, get_object_or_404
from django.db.models import Case, When, Value, IntegerField
from django.utils.text import slugify
from django.core.paginator import Paginator
import json
from manga.models import Manga, Chapter
import requests
import re
from bs4 import BeautifulSoup

header = {'User-agent': 'Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36'}

def extract_chapter_number(title):
    match = re.search(r'chapter\s*([\d.]+)', title, re.IGNORECASE)
    if match:
        return match.group(1)
    match = re.search(r'([\d.]+)', title)
    return match.group(1) if match else "0"

def save_chapters(manga, scraped_data):
    for entry in scraped_data:
        for url, title in entry.items():
            chapter_number = extract_chapter_number(title)
            mangaowl_url = url

            chapter, created = Chapter.objects.get_or_create(
                manga=manga,
                chapter_number=chapter_number,
                defaults={
                    'title': title,
                    'slug': slugify(f"{title}"),
                    'image_urls': [],  # You can fill this when scraping images
                    'mangaowl_url': mangaowl_url,
                }
            )

            if created:
                print(f"New Chapter saved: {chapter}")

def scrape_chapter(slug):
    url = "https://mangaowl.io/read-1/" + slug + "/ajax/chapters/"
    chapter_links = requests.post(url, headers=header)
    lik_html = BeautifulSoup(chapter_links.text, 'lxml')
    chap = lik_html.find("ul")
    chap_links = [{c['href']: c.text.strip()} for c in chap.find_all("a")]
    seen = set()
    cleaned = []

    for entry in chap_links:
        for url, title in entry.items():
            if title and url not in seen:
                cleaned.append({url: title})
                seen.add(url)
    return cleaned

def scrape_images(chapter_url):
    chapter_html = requests.get(chapter_url, headers=header)
    chapter_soup  = BeautifulSoup(chapter_html.text, 'lxml')
    links_group = chapter_soup.select("div.page-break.no-gaps")
    img_links = [i.img['data-src'].strip() for i in links_group]
    # print(img_links)
    return img_links




# Home page 
def home(request):
    # mangas = Manga.objects.all().order_by('title')
    mangas = Manga.objects.annotate(
        has_image=Case(
            When(image_url__isnull=False, then=Value(1)),
            default=Value(0),
            output_field=IntegerField()
        )
    ).order_by('-has_image', 'title')    
    
    paginator = Paginator(mangas, 20)
    page_number = request.GET.get('page')
    page_obj = paginator.get_page(page_number)
    return render(request, 'index.html', {'mangas': page_obj})

# Manga Detail View
def manga_detail_view(request, slug):
    manga = get_object_or_404(Manga, slug=slug)


    if not manga.chapters.exists():
        # print("No chapters found in DB — scraping...")
        scraped_chapters = scrape_chapter(slug)
        save_chapters(manga, scraped_chapters)
    else:
        print("Chapters already exist")

    chapters = manga.chapters.order_by('-chapter_number')
    


    return render(request, 'manga_detail.html', 
                  {
                'manga': manga, 
                'chapters': chapters, 
                'meta_description': f"Read {manga.title} online, updated daily.",
                'meta_keywords': ', '.join(manga.genre),
                'genre_json': json.dumps(manga.genre),
        })

# Chapter Detail View
def chapter_detail_view(request, manga_slug, chapter_number):
    # Get the manga object by slug
    manga = get_object_or_404(Manga, slug=manga_slug)
    # print(manga_slug)

    # Get the chapter object by chapter_number and manga
    chapter = get_object_or_404(Chapter, manga=manga, slug=chapter_number)

    if not chapter.image_urls:
        print("No image URLs found")
        image_urls = scrape_images(chapter.mangaowl_url)
        chapter.image_urls = image_urls
        chapter.save(update_fields=['image_urls'])
    else:
        print("Image already exist")

    next_chapter = manga.chapters.filter(chapter_number__gt=chapter.chapter_number).order_by('chapter_number').first()
    prev_chapter = manga.chapters.filter(chapter_number__lt=chapter.chapter_number).order_by('-chapter_number').first()

    # Pass the chapter to the template
    return render(request, 'chapter_detail.html', {
        'manga': manga,
        'chapter': chapter,
        'next_chapter': next_chapter,
        'prev_chapter': prev_chapter,
        'meta_description': f"Read {manga.title} {chapter.title} online. Updated daily on MangaFoxy.",
        'meta_keywords' : ', '.join(manga.genre + [manga.title, chapter.title]),

    })
