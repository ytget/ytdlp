# Homebrew Integration - Deployment Instructions

## What was implemented

1. ✅ Created Homebrew Formula (`Formula/ytdlp.rb`)
2. ✅ Set up GitHub Actions workflow for automatic releases (`.github/workflows/release.yml`)
3. ✅ Updated README.md with Homebrew installation instructions
4. ✅ Created separate Homebrew tap (`homebrew-tap/`)
5. ✅ Tested formula correctness

## Next steps for deployment

### 1. Create release
```bash
# Create release tag
git tag v2.0.1
git push origin v2.0.1
```

### 2. Create Homebrew Tap repository
Create new repository on GitHub: `ytget/homebrew-ytdlp`

Copy contents of `homebrew-tap/` to the new repository.

### 3. Update formula with correct SHA256
After creating release, get SHA256 of archive:
```bash
curl -L https://github.com/ytget/ytdlp/archive/v2.0.1.tar.gz | shasum -a 256
```

Update `sha256` field in files:
- `Formula/ytdlp.rb`
- `homebrew-tap/ytdlp.rb`

### 4. Submit formula to Homebrew Core (optional)
To add to main Homebrew repository:
```bash
brew create https://github.com/ytget/ytdlp/archive/v2.0.1.tar.gz --tap=homebrew/core
```

## Usage

After deployment users will be able to install via:

```bash
# From main repository (if accepted to core)
brew install ytdlp-go

# From our tap
brew tap ytget/ytdlp
brew install ytdlp-go

# Directly from repository
brew install ytget/ytdlp/Formula/ytdlp-go
```

## Files to commit

Add to repository:
- `Formula/ytdlp.rb`
- `.github/workflows/release.yml`
- `homebrew-tap/` (directory)
- Updated `README.md`
