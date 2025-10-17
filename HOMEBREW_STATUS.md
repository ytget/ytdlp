# Homebrew Integration Status - Verification

## ✅ What we have:

1. **Homebrew Formula** - `Formula/ytdlp.rb` with correct SHA256 for release v2.0.0
2. **GitHub Actions workflow** - `.github/workflows/release.yml` for automatic releases
3. **Homebrew Tap** - `homebrew-tap/` directory with formula and instructions
4. **Updated README** - with correct installation commands
5. **Existing releases** - tags v0.1.0 and v2.0.0 already created

## ⚠️ What needs to be done:

### 1. Create separate repository for Homebrew Tap
```bash
# Create new repository on GitHub: ytget/homebrew-ytdlp
# Copy contents of homebrew-tap/ to new repository
```

### 2. Create new release with binary files
```bash
# Create new tag (e.g., v2.0.1)
git tag v2.0.1
git push origin v2.0.1
# GitHub Actions will automatically create release with binary files
```

### 3. Update formulas with new SHA256
After creating new release update SHA256 in:
- `Formula/ytdlp.rb`
- `homebrew-tap/ytdlp.rb`

## 🔧 Current installation commands:

### Works now:
```bash
# Install directly from repository (after commit)
brew install ytget/ytdlp/Formula/ytdlp-go
```

### Will work after creating tap:
```bash
# Install via tap
brew tap ytget/ytdlp
brew install ytdlp-go
```

## 📋 Action plan:

1. **Now**: Commit changes to main repository
2. **Next step**: Create repository `ytget/homebrew-ytdlp`
3. **After that**: Create new release with binary files
4. **Finally**: Update formulas with new SHA256

## ✅ Verified:

- Formula is correct and passes dry-run test
- GitHub Actions workflow is configured properly
- SHA256 for v2.0.0 is correct: `c9c9214e2d563e833eb0f079d876687a72762358bb6040d045173d62978e5c6b`
- Installation commands in README are correct
