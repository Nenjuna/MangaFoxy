# manga/views.py
from rest_framework import status
from rest_framework.response import Response
from rest_framework.decorators import api_view
from .models import Manga, Chapter
from .serializers import MangaSerializer, ChapterSerializer


@api_view(['GET', 'POST'])
def manga_list(request):
    """
    List all mangas, or create a new manga.
    """
    if request.method == 'GET':
        mangas = Manga.objects.all()
        serializer = MangaSerializer(mangas, many=True)
        return Response(serializer.data)

    elif request.method == 'POST':
        serializer = MangaSerializer(data=request.data)
        if serializer.is_valid():
            serializer.save()
            return Response(serializer.data, status=status.HTTP_201_CREATED)
        return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)


@api_view(['GET', 'PUT', 'DELETE'])
def manga_detail(request, slug):
    """
    Retrieve, update or delete a manga.
    """
    try:
        manga = Manga.objects.get(slug=slug)
    except Manga.DoesNotExist:
        return Response(status=status.HTTP_404_NOT_FOUND)

    if request.method == 'GET':
        serializer = MangaSerializer(manga)
        return Response(serializer.data)

    elif request.method == 'PUT':
        serializer = MangaSerializer(manga, data=request.data)
        if serializer.is_valid():
            serializer.save()
            return Response(serializer.data)
        return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)

    elif request.method == 'DELETE':
        manga.delete()
        return Response(status=status.HTTP_204_NO_CONTENT)


@api_view(['GET', 'POST'])
def chapter_list_create(request):
    """
    GET: List all chapters
    POST: Create a new chapter
    """
    if request.method == 'GET':
        chapters = Chapter.objects.all()
        serializer = ChapterSerializer(chapters, many=True)
        return Response(serializer.data)

    elif request.method == 'POST':
        serializer = ChapterSerializer(data=request.data)
        if serializer.is_valid():
            serializer.save()  # Calls your model's save method (auto-slug etc.)
            return Response(serializer.data, status=status.HTTP_201_CREATED)
        return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)