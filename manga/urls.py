# manga/urls.py
from django.urls import path
from . import views

urlpatterns = [
    path('mangas/', views.manga_list, name='manga-list'),
    path('mangas/<slug:slug>/', views.manga_detail, name='manga-detail'),
    path('chapters/', views.chapter_list_create, name='chapter-list-create'),
]
