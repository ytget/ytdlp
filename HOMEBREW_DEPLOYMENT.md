# Homebrew Integration - Deployment Instructions

## Что было сделано

1. ✅ Создана Homebrew Formula (`Formula/ytdlp.rb`)
2. ✅ Настроен GitHub Actions workflow для автоматических релизов (`.github/workflows/release.yml`)
3. ✅ Обновлен README.md с инструкциями по установке через Homebrew
4. ✅ Создан отдельный Homebrew tap (`homebrew-tap/`)
5. ✅ Протестирована корректность формулы

## Следующие шаги для развертывания

### 1. Создание релиза
```bash
# Создать тег для релиза
git tag v2.0.0
git push origin v2.0.0
```

### 2. Создание Homebrew Tap репозитория
Создать новый репозиторий на GitHub: `ytget/homebrew-ytdlp`

Скопировать содержимое `homebrew-tap/` в новый репозиторий.

### 3. Обновление формулы с правильным SHA256
После создания релиза, получить SHA256 архива:
```bash
curl -L https://github.com/ytget/ytdlp/archive/v2.0.0.tar.gz | shasum -a 256
```

Обновить поле `sha256` в файлах:
- `Formula/ytdlp.rb`
- `homebrew-tap/ytdlp.rb`

### 4. Подача формулы в Homebrew Core (опционально)
Для добавления в основной репозиторий Homebrew:
```bash
brew create https://github.com/ytget/ytdlp/archive/v2.0.0.tar.gz --tap=homebrew/core
```

## Использование

После развертывания пользователи смогут устанавливать через:

```bash
# Из основного репозитория (если принято в core)
brew install ytdlp-go

# Из нашего tap
brew tap ytget/ytdlp
brew install ytdlp-go

# Прямо из репозитория
brew install ytget/ytdlp/Formula/ytdlp-go
```

## Файлы для коммита

Добавить в репозиторий:
- `Formula/ytdlp.rb`
- `.github/workflows/release.yml`
- `homebrew-tap/` (директория)
- Обновленный `README.md`
