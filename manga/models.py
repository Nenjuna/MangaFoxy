from django.db import models
from django.utils.text import slugify
from django.core.validators import MinValueValidator, MaxValueValidator
import re


#Function to extact Chapter number
def extract_chapter_number(title):
    match = re.search(r'chapter\s*([\d.]+)', title, re.IGNORECASE)
    if match:
        return match.group(1)
    # fallback: get any number
    match = re.search(r'([\d.]+)', title)
    return match.group(1) if match else None


class Manga(models.Model):
    STATUS_CHOICES = [
        ('ongoing', 'Ongoing'),
        ('completed', 'Completed'),
        ('hiatus', 'Hiatus'),
    ]

    title = models.CharField(max_length=255)
    alternative = models.CharField(
        max_length=255, blank=True, null=True)  # optional
    slug = models.SlugField(unique=True, blank=True)

    status = models.CharField(max_length=10, choices=STATUS_CHOICES)

    genre = models.JSONField(default=list)  # storing array as JSON

    rating = models.FloatField(
        null=True, blank=True,
        validators=[MinValueValidator(0.0), MaxValueValidator(10.0)]
    )

    rank = models.PositiveIntegerField(null=True, blank=True)

    year = models.PositiveIntegerField(null=True, blank=True)  # optional year

    image_url = models.TextField(blank=True, null=True)

    last_updated_chapter = models.CharField(
        max_length=100,
        null=True,
        blank=True,
        help_text="ID of the last updated chapter"
    )

    summary = models.TextField(blank=True, null=True)

    chapter = models.JSONField(default=list, blank=True, null=True)

    last_scraped = models.DateTimeField(null=True, blank=True)


    def save(self, *args, **kwargs):
        if not self.slug:
            self.slug = slugify(self.title)
        super().save(*args, **kwargs)

    def __str__(self):
        return self.title

class Chapter(models.Model):
    manga = models.ForeignKey(Manga, related_name='chapters', on_delete=models.CASCADE)
    title = models.CharField(max_length=255)
    subtitle = models.CharField(max_length=255, blank=True, null=True)
    chapter_number = models.CharField(max_length=50)
    slug = models.SlugField()
    image_urls = models.JSONField(default=list, blank=True, null=True)  # Store image URLs as a list
    created_at = models.DateTimeField(auto_now_add=True)
    mangaowl_url = models.URLField(blank=True, null=True)

    class Meta:
        unique_together = ('manga', 'chapter_number')

    def save(self, *args, **kwargs):
        if not self.chapter_number:
            self.chapter_number = extract_chapter_number(self.title) or "0"
        self.slug = slugify(f"{self.title}")
        super().save(*args, **kwargs)

    def __str__(self):
        return f"{self.manga.title} - {self.title}"