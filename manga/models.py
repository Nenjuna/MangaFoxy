from django.db import models
from django.utils.text import slugify
from django.core.validators import MinValueValidator, MaxValueValidator
from django.contrib.contenttypes.models import ContentType
from django.contrib.contenttypes.fields import GenericForeignKey
from django.utils.timezone import now
from django.templatetags.static import static

import re


# Function to extact Chapter number
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

    view_count = models.PositiveIntegerField(default=0)
    updated_at = models.DateTimeField(auto_now=True)

    def save(self, *args, **kwargs):
        if not self.slug:
            self.slug = slugify(self.title)
        super().save(*args, **kwargs)

    def __str__(self):
        return self.title

    # @property
    # def view_count(self):
    #     content_type = ContentType.objects.get_for_model(self)
    #     return ViewLog.objects.filter(content_type=content_type, object_id=self.id).count()

    @property
    def unique_view_count(self):
        content_type = ContentType.objects.get_for_model(self)
        return ViewLog.objects.filter(content_type=content_type, object_id=self.id).values('session_id', 'ip_address').distinct().count()

    @property
    def thumbnail_url(self):
        if self.image_url and self.image_url.strip() != "":
            return self.image_url
        return static('images/no-thumbnail.png')


class Chapter(models.Model):
    manga = models.ForeignKey(
        Manga, related_name='chapters', on_delete=models.CASCADE)
    title = models.CharField(max_length=255)
    subtitle = models.CharField(max_length=255, blank=True, null=True)
    chapter_number = models.CharField(max_length=50)
    slug = models.SlugField()
    # Store image URLs as a list
    image_urls = models.JSONField(default=list, blank=True, null=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    mangaowl_url = models.URLField(blank=True, null=True)
    view_count = models.PositiveIntegerField(default=0)

    class Meta:
        unique_together = ('manga', 'chapter_number')

    def save(self, *args, **kwargs):
        if not self.chapter_number:
            self.chapter_number = extract_chapter_number(self.title) or "0"
        self.slug = slugify(f"{self.title}")
        super().save(*args, **kwargs)

    def __str__(self):
        return f"{self.manga.title} - {self.title}"

    # @property
    # def view_count(self):
    #     content_type = ContentType.objects.get_for_model(self)
    #     return ViewLog.objects.filter(content_type=content_type, object_id=self.id).count()

    @property
    def unique_view_count(self):
        content_type = ContentType.objects.get_for_model(self)
        return ViewLog.objects.filter(content_type=content_type, object_id=self.id).values('session_id', 'ip_address').distinct().count()


class ViewLog(models.Model):
    ip_address = models.GenericIPAddressField(null=True, blank=True)
    session_id = models.CharField(max_length=100, null=True, blank=True)
    timestamp = models.DateTimeField(default=now)
    processed = models.BooleanField(default=False)

    # Generic relation to Manga or Chapter
    content_type = models.ForeignKey(ContentType, on_delete=models.CASCADE)
    object_id = models.PositiveIntegerField()
    content_object = GenericForeignKey()

    class Meta:
        indexes = [
            models.Index(fields=['content_type', 'object_id', 'timestamp']),
            models.Index(fields=['processed']),
            models.Index(fields=['timestamp'])
        ]
        # unique_together = ('content_type', 'object_id', 'ip_address', 'session_id')

    def __str__(self):
        return f"{self.content_object} viewed at {self.timestamp}"
