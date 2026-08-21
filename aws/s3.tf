resource "aws_s3_bucket" "immich_backup" {
  bucket = "immich-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "immich_backup" {
  bucket = aws_s3_bucket.immich_backup.id

  # DB dumps: keep 30 days only
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }

}

resource "aws_s3_bucket_public_access_block" "immich_backup" {
  bucket = aws_s3_bucket.immich_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Seafile backup
resource "aws_s3_bucket" "seafile_backup" {
  bucket = "seafile-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "seafile_backup" {
  bucket = aws_s3_bucket.seafile_backup.id

  # DB dumps: keep 30 days only
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }

}

resource "aws_s3_bucket_public_access_block" "seafile_backup" {
  bucket = aws_s3_bucket.seafile_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Health Hub backup
resource "aws_s3_bucket" "health_backup" {
  bucket = "health-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "health_backup" {
  bucket = aws_s3_bucket.health_backup.id

  # DB dumps: keep 30 days only
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }
}

resource "aws_s3_bucket_public_access_block" "health_backup" {
  bucket = aws_s3_bucket.health_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Minecraft backup
# No lifecycle rule — restic manages its own retention via `restic forget/prune`
# (pruneResticRetention in k8s/minecraft/values.yaml). Storage class: Standard
# (default) — backup dataset is small, IT per-object monitoring cost not worth it.
resource "aws_s3_bucket" "minecraft_backup" {
  bucket = "minecraft-backup-json-server"
}

resource "aws_s3_bucket_public_access_block" "minecraft_backup" {
  bucket = aws_s3_bucket.minecraft_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# AMANG production backup (postgres 논리 덤프 + MinIO 오브젝트 미러)
resource "aws_s3_bucket" "amang_backup" {
  bucket = "amang-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "amang_backup" {
  bucket = aws_s3_bucket.amang_backup.id

  # DB dumps: keep 30 days only.
  # media/ 는 MinIO 미러라 만료 규칙을 두지 않는다 (immich/seafile 과 동일).
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }
}

resource "aws_s3_bucket_public_access_block" "amang_backup" {
  bucket = aws_s3_bucket.amang_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
