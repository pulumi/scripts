# publish.ps1 will publish the current bits from the usual build location to an S3 build share. See
# Print-Usage for usage information
param (
  $PublishFile, 
  $ReleaseName,
  [parameter(ValueFromRemainingArguments=$true)]$PublishVersions
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = "Stop"

function Print-Usage() {
    Write-Host "usage: publish.ps1 FILE RELEASE VERSION..."
    Write-Host "   where FILE is the release package to publish"
    Write-Host "         RELEASE is something like pulumi-fabric, etc."
    Write-Host "         VERSION is one or more commitishes, tags, etc. for this release"
}

if (!$PublishFile) {
    Write-Host "error: missing the file to publish"
    Print-Usage
    exit 1
}

if (!$ReleaseName) {
    Write-Host "error: missing the name of the release to publish"
    Print-Usage
    exit 1
}

if (!$PublishVersions -or $PublishVersions.Count -le 0) {
    Write-Host "error: missing the name of the release to publish"
    Print-Usage
    exit 1
}

$PublishPrefix="s3://eng.pulumi.com/releases/${ReleaseName}"

$firstTarget = $null
foreach ($publishVersion in $PublishVersions) {
    $publishTarget = "${PublishPrefix}/${publishVersion}.zip"

    Write-Host "Publishing ${ReleaseName}@${publishVersion} to: $publishTarget"

    if ($firstTarget -eq $null) {
        # Upload the first one for real.
        aws s3 cp "${PublishFile}" "${PublishTarget}" --acl bucket-owner-full-control
        $firstTarget = $publishTarget
    } else {
        # For all others, reuse the first target to avoid re-uploading.
        aws s3 cp "${firstTarget}" "${PublishTarget}" --acl bucket-owner-full-control
    }
}
