from django.contrib import admin
from .models import Manga, Chapter, ViewLog, Update


@admin.register(Update)
class UpdateAdmin(admin.ModelAdmin):
    list_display = ('title', 'update_type', 'created_at', 'manga', 'chapter')
    list_filter = ('update_type', 'created_at')
    search_fields = ('title', 'content', 'manga__title', 'chapter__title')
    date_hierarchy = 'created_at'
    readonly_fields = ('created_at', 'updated_at')
    fieldsets = (
        (None, {
            'fields': ('title', 'content', 'update_type')
        }),
        ('Related Content', {
            'fields': ('manga', 'chapter'),
            'classes': ('collapse',)
        }),
        ('Timestamps', {
            'fields': ('created_at', 'updated_at'),
            'classes': ('collapse',)
        }),
    )

    def get_queryset(self, request):
        return super().get_queryset(request).select_related('manga', 'chapter')
