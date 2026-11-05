& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $repository = 'DasDarki/nwm'
    $installDir = if ($env:NWM_INSTALL_DIR) { $env:NWM_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\nwm' }
    $version = if ($env:NWM_VERSION) { $env:NWM_VERSION } else { 'latest' }

    function Write-Step([string] $message) {
        Write-Host '==> ' -ForegroundColor Magenta -NoNewline
        Write-Host $message
    }

    $processorArchitecture = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    $architecture = switch ($processorArchitecture) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { throw "Unsupported architecture $processorArchitecture" }
    }

    $asset = "nwm_windows_$architecture.exe"
    if ($version -eq 'latest') {
        $baseUrl = "https://github.com/$repository/releases/latest/download"
    } else {
        if (-not $version.StartsWith('v')) { $version = "v$version" }
        $baseUrl = "https://github.com/$repository/releases/download/$version"
    }

    $temporary = Join-Path ([IO.Path]::GetTempPath()) ("nwm-install-" + [Guid]::NewGuid())
    New-Item -ItemType Directory -Path $temporary | Out-Null
    try {
        Write-Step "Downloading $asset ($version)"
        $binary = Join-Path $temporary 'nwm.exe'
        $checksums = Join-Path $temporary 'checksums.txt'
        Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $binary -UseBasicParsing
        Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksums -UseBasicParsing

        $expected = $null
        foreach ($line in Get-Content $checksums) {
            $fields = $line -split '\s+'
            if ($fields.Count -ge 2 -and $fields[1] -eq $asset) { $expected = $fields[0].ToLowerInvariant() }
        }
        if (-not $expected) { throw "checksums.txt has no entry for $asset" }
        $actual = (Get-FileHash -Algorithm SHA256 -Path $binary).Hash.ToLowerInvariant()
        if ($actual -ne $expected) { throw 'Checksum mismatch, refusing to install' }

        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        $target = Join-Path $installDir 'nwm.exe'
        if (Test-Path $target) {
            $previous = "$target.old"
            Remove-Item $previous -Force -ErrorAction SilentlyContinue
            Move-Item $target $previous -Force
        }
        Move-Item $binary $target -Force
        Write-Step "Installed nwm to $target"
    } finally {
        Remove-Item $temporary -Recurse -Force -ErrorAction SilentlyContinue
    }

    $environmentKey = Get-Item -Path 'HKCU:\Environment'
    $rawPath = $environmentKey.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    $entries = @($rawPath -split ';' | Where-Object { $_ })
    $expandedEntries = @($entries | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') })
    if ($expandedEntries -notcontains $installDir.TrimEnd('\')) {
        Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value (($entries + $installDir) -join ';') -Type ExpandString
        [Environment]::SetEnvironmentVariable('NWM_INSTALLER_REFRESH', '1', 'User')
        [Environment]::SetEnvironmentVariable('NWM_INSTALLER_REFRESH', $null, 'User')
        Write-Step "Added $installDir to your user PATH"
    }
    if (($env:Path -split ';') -notcontains $installDir) {
        $env:Path = "$env:Path;$installDir"
    }

    & (Join-Path $installDir 'nwm.exe') version
    Write-Step 'Done, run: nwm --help (open a new terminal if the command is not found)'
}
