from django.shortcuts import render, get_object_or_404
from django.db.models import Case, When, Value, IntegerField, Q, FloatField
from django.db.models.functions import Cast
from django.utils.text import slugify
from django.core.paginator import Paginator
import json
from manga.models import Manga, Chapter, Update
from django.utils import timezone
from manga.models import Manga, Chapter


import requests
import re
from bs4 import BeautifulSoup
from .session_logger import log_view
from django.http import Http404, HttpResponse
from django.template.loader import render_to_string
from django.urls import reverse
from django.template import loader
from django.views.decorators.cache import cache_page
from django.db.models import Max


header = {
    'User-agent': 'Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36'}


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
    print(chapter_url)
    # print(chapter_html.text)
    chapter_soup = BeautifulSoup(chapter_html.text, 'lxml')
    links_group = chapter_soup.select("div.page-break.no-gaps")
    img_links = []
    for i in links_group:
        img_tag = i.img
        if img_tag:
            src = img_tag.get('data-src') or img_tag.get('src')
            if src:
                img_links.append(src.strip())
    # print(img_links)
    return img_links


# Home page
def home(request):
    mangas = Manga.objects.annotate(
        # total_views=Subquery(view_count_subquery, output_field=IntegerField()),
        has_image=Case(
            When(~Q(image_url__isnull=True) & ~Q(image_url=''), then=Value(1)),
            default=Value(0),
            output_field=IntegerField()
        )
    ).order_by('-view_count', '-has_image', 'title')

    paginator = Paginator(mangas, 32)
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

    chapters = manga.chapters.annotate(chapter_number_float=Cast(
        'chapter_number', FloatField())).order_by('-chapter_number_float')

    first_chapter = chapters.last()  # lowest chapter number
    latest_chapter = chapters.first()  # highest chapter number

    log_view(request, manga)  # logging manga details

    return render(request, 'manga_detail.html',
                  {
                      'manga': manga,
                      'chapters': chapters,
                      'meta_description': f"Read {manga.title} online, updated daily.",
                      'meta_keywords': ', '.join(manga.genre),
                      'genre_json': json.dumps(manga.genre),
                      'first_chapter': first_chapter,
                      'latest_chapter': latest_chapter,
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

    next_chapter = manga.chapters.filter(
        chapter_number__gt=chapter.chapter_number).order_by('chapter_number').first()
    prev_chapter = manga.chapters.filter(
        chapter_number__lt=chapter.chapter_number).order_by('-chapter_number').first()

    log_view(request, chapter)  # logging for chapter

    # Pass the chapter to the template
    return render(request, 'chapter_detail.html', {
        'manga': manga,
        'chapter': chapter,
        'next_chapter': next_chapter,
        'prev_chapter': prev_chapter,
        'meta_description': f"Read {manga.title} {chapter.title} online. Updated daily on MangaFoxy.",
        'meta_keywords': ', '.join(manga.genre + [manga.title, chapter.title]),
    })


def genre_view(request, genre_slug=None):
    # Get all unique genres from the database
    all_genres = set()
    for manga in Manga.objects.all():
        all_genres.update(manga.genre)
    genres = sorted(list(all_genres))

    # If genre_slug is provided, use it as the selected genre
    selected_genre = genre_slug.replace(
        '-', ' ').title() if genre_slug else None

    # Filter mangas by genre if one is selected
    if selected_genre:
        mangas = Manga.objects.filter(genre__icontains=selected_genre)
    else:
        # For the genre list page, show all manga
        mangas = Manga.objects.all()

    # Pagination
    paginator = Paginator(mangas, 24)  # Show 24 mangas per page
    page = request.GET.get('page')
    mangas = paginator.get_page(page)

    # If we're on a genre detail page and the genre doesn't exist, return 404
    if genre_slug and not mangas:
        raise Http404("Genre not found")

    return render(request, 'genre.html', {
        'genres': genres,
        'selected_genre': selected_genre,
        'mangas': mangas,
    })


def copyright_view(request):
    return render(request, 'copyright.html')


def terms_view(request):
    return render(request, 'terms.html')


@cache_page(60 * 60)  # Cache for 1 hour
def sitemap_index(request):
    # Calculate total number of chapter pages
    total_chapters = Chapter.objects.count()
    items_per_page = 1000
    total_pages = (total_chapters + items_per_page - 1) // items_per_page
    chapter_pages = range(1, total_pages + 1)

    template = loader.get_template('sitemap_index.xml')
    return HttpResponse(template.render({'chapter_pages': chapter_pages}, request), content_type='application/xml')


@cache_page(60 * 60)  # Cache for 1 hour
def sitemap_manga(request):
    page = request.GET.get('page', 1)
    mangas = Manga.objects.all().order_by('-updated_at')
    paginator = Paginator(mangas, 1000)  # 1000 URLs per sitemap

    try:
        mangas = paginator.page(page)
    except:
        mangas = paginator.page(1)

    template = loader.get_template('sitemap-manga.xml')
    return HttpResponse(template.render({'mangas': mangas}, request), content_type='application/xml')


@cache_page(60 * 60)  # Cache for 1 hour
def sitemap_chapters(request, page=1):
    chapters = Chapter.objects.select_related(
        'manga').all().order_by('-created_at')
    paginator = Paginator(chapters, 1000)  # 1000 URLs per sitemap

    try:
        chapters = paginator.page(page)
    except:
        raise Http404("Page not found")

    template = loader.get_template('sitemap-chapters.xml')
    return HttpResponse(template.render({'chapters': chapters}, request), content_type='application/xml')


@cache_page(60 * 60)  # Cache for 1 hour
def sitemap_genres(request):
    # Get all unique genres
    genres = Manga.objects.values_list('genre', flat=True).distinct()
    unique_genres = set()
    for genre_list in genres:
        if genre_list:
            unique_genres.update(genre_list)
    genres = sorted(list(unique_genres))

    template = loader.get_template('sitemap-genres.xml')
    return HttpResponse(template.render({'genres': genres}, request), content_type='application/xml')


@cache_page(60 * 60)  # Cache for 1 hour
def sitemap_pages(request):
    template = loader.get_template('sitemap-pages.xml')
    return HttpResponse(template.render({}, request), content_type='application/xml')


def about_view(request):
    return render(request, 'about.html')


def contact_view(request):
    if request.method == 'POST':
        # Here you can add logic to handle the contact form submission
        # For example, sending an email or saving to database
        name = request.POST.get('name')
        email = request.POST.get('email')
        subject = request.POST.get('subject')
        message = request.POST.get('message')

        # Add your email sending logic here
        # For now, we'll just render the page with a success message
        return render(request, 'contact.html', {'success': True})

    return render(request, 'contact.html')


def updates_view(request):
    updates = Update.objects.all().order_by('-created_at')
    paginator = Paginator(updates, 20)  # Show 20 updates per page
    page = request.GET.get('page')
    updates = paginator.get_page(page)

    return render(request, 'updates.html', {
        'updates': updates,
        'meta_description': 'Latest updates on new manga releases, chapter updates, and site news.',
        'meta_keywords': 'manga updates, new chapters, new manga, site news',
    })
