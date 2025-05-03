from django.contrib.contenttypes.models import ContentType
from django.db.models import Count
from .models import Manga, ViewLog

def update_manga_view_counts():
    manga_type = ContentType.objects.get_for_model(Manga)

    view_counts = (
        ViewLog.objects.filter(content_type=manga_type)
        .values('object_id')
        .annotate(total=Count('id'))
    )

    for entry in view_counts:
        # Manga.objects.filter(id=entry['object_id']).update(view_count=entry['total'])
        print(entry)
