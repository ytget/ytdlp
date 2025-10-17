class YtdlpGo < Formula
  desc "Native Go library and CLI to download online videos — no external binaries, Android-friendly"
  homepage "https://github.com/ytget/ytdlp"
  url "https://github.com/ytget/ytdlp/archive/v2.0.0.tar.gz"
  sha256 "c9c9214e2d563e833eb0f079d876687a72762358bb6040d045173d62978e5c6b"
  license "MIT"
  head "https://github.com/ytget/ytdlp.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", "ytdlp", "./cmd/ytdlp"
    bin.install "ytdlp"
  end

  test do
    # Test that the binary was installed correctly
    assert_match "ytdlp", shell_output("#{bin}/ytdlp --help", 1)
  end
end
