resource "aws_iam_user" "immich_backup" {
  name = "immich-backup"
}

resource "aws_iam_policy" "immich_backup" {
  name        = "immich-backup-s3"
  description = "Allow Immich backup CronJobs to sync to S3"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject",
      ]
      Resource = [
        aws_s3_bucket.immich_backup.arn,
        "${aws_s3_bucket.immich_backup.arn}/*",
      ]
    }]
  })
}

resource "aws_iam_user_policy_attachment" "immich_backup" {
  user       = aws_iam_user.immich_backup.name
  policy_arn = aws_iam_policy.immich_backup.arn
}

resource "aws_iam_access_key" "immich_backup" {
  user = aws_iam_user.immich_backup.name
}
