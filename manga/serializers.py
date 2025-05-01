# manga/serializers.py
from rest_framework import serializers
from .models import Manga, Chapter


class MangaSerializer(serializers.ModelSerializer):
    class Meta:
        model = Manga
        fields = '__all__'  # Include all fields from the Manga model


class ChapterSerializer(serializers.ModelSerializer):
    class Meta:
        model = Chapter
        fields = '__all__'