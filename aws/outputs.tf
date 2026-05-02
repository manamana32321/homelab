output "immich_backup_access_key_id" {
  description = "Access Key ID for immich-backup IAM user"
  value       = aws_iam_access_key.immich_backup.id
}

output "immich_backup_secret_access_key" {
  description = "Secret Access Key for immich-backup IAM user"
  value       = aws_iam_access_key.immich_backup.secret
  sensitive   = true
}

output "immich_backup_bucket" {
  description = "S3 bucket name for Immich backups"
  value       = aws_s3_bucket.immich_backup.bucket
}

# Seafile backup
output "seafile_backup_access_key_id" {
  description = "Access Key ID for seafile-backup IAM user"
  value       = aws_iam_access_key.seafile_backup.id
}

output "seafile_backup_secret_access_key" {
  description = "Secret Access Key for seafile-backup IAM user"
  value       = aws_iam_access_key.seafile_backup.secret
  sensitive   = true
}

output "seafile_backup_bucket" {
  description = "S3 bucket name for Seafile backups"
  value       = aws_s3_bucket.seafile_backup.bucket
}

# Health Hub backup
output "health_backup_access_key_id" {
  description = "Access Key ID for health-backup IAM user"
  value       = aws_iam_access_key.health_backup.id
}

output "health_backup_secret_access_key" {
  description = "Secret Access Key for health-backup IAM user"
  value       = aws_iam_access_key.health_backup.secret
  sensitive   = true
}

output "health_backup_bucket" {
  description = "S3 bucket name for Health Hub backups"
  value       = aws_s3_bucket.health_backup.bucket
}

# Probe user (read-only diagnostics)
output "probe_access_key_id" {
  description = "Access Key ID for homelab-probe IAM user (read-only diagnostics)"
  value       = aws_iam_access_key.probe.id
}

output "probe_secret_access_key" {
  description = "Secret Access Key for homelab-probe IAM user"
  value       = aws_iam_access_key.probe.secret
  sensitive   = true
}
