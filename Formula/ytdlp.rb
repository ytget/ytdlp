class Ytdlp < Formula
  desc "Native Go library and CLI to download online videos — no external binaries, Android-friendly"
  homepage "https://github.com/ytget/ytdlp"
  url "https://github.com/ytget/ytdlp/archive/v2.0.3.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  head "https://github.com/ytget/ytdlp.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "mod", "download"
    system "go", "build", "-o", "ytdlp-bin", "./cmd/ytdlp"
    bin.install "ytdlp-bin" => "ytdlp"
  end

  test do
    # Test that the binary was installed correctly
    assert_match "ytdlp", shell_output("#{bin}/ytdlp --help", 1)
  end
end
