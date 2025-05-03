from django.core.management.base import BaseCommand
from django.contrib.contenttypes.models import ContentType
from manga.models import Manga, ViewLog, Chapter # Replace with your actual app name
from django.db.models import Count

class Command(BaseCommand):
    help = "Update Manga view_count field from ViewLog"

    def handle(self, *args, **kwargs):
        manga_type = ContentType.objects.get_for_model(Manga)
        chapter_type = ContentType.objects.get_for_model(Chapter)

        self.stdout.write(self.style.SUCCESS("Manga view counts updated."))

        view_counts = (
            ViewLog.objects.filter(content_type=manga_type)
            .values('object_id')
            .annotate(total=Count('id'))
        )

        chapter_counts = (
            ViewLog.objects.filter(content_type=chapter_type).values('object_id').annotate(total=Count('id'))
        )

        for entry in view_counts:
            Manga.objects.filter(id=entry['object_id']).update(view_count=entry['total'])
            print(entry)

        self.stdout.write(self.style.SUCCESS("Manga view counts updated."))
        self.stdout.write(self.style.SUCCESS("Chapter view counts started."))

        for entry in chapter_counts:
            Chapter.objects.filter(id=entry['object_id']).update(view_count=entry['total'])
            print(entry)

        self.stdout.write(self.style.SUCCESS("Chapter view counts updated."))
