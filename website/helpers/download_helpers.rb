require "net/http"

$terraform_files = {}
$terraform_os = []

if ENV["TERRAFORM_VERSION"]
  raise "BINTRAY_API_KEY must be set." if !ENV["BINTRAY_API_KEY"]
  http = Net::HTTP.new("dl.bintray.com", 80)
  req = Net::HTTP::Get.new("/mitchellh/terraform/")
  req.basic_auth "mitchellh", ENV["BINTRAY_API_KEY"]
  response = http.request(req)

  response.body.split("\n").each do |line|
    next if line !~ /\/mitchellh\/terraform\/terraform_(#{Regexp.quote(ENV["TERRAFORM_VERSION"])}.+?)'/
    filename = $1.to_s
    os = filename.split("_")[1]
    next if os == "SHA256SUMS"
    next if os == "web"

    $terraform_files[os] ||= []
    $terraform_files[os] << "terraform_#{filename}"
  end

  $terraform_os = ["darwin", "linux", "windows"] & $terraform_files.keys
  $terraform_os += $terraform_files.keys
  $terraform_os.uniq!

  $terraform_files.each do |key, value|
    value.sort!
  end
end

module DownloadHelpers
  def download_arch(file)
    parts = file.split("_")
    return "" if parts.length != 4
    parts[3].split(".")[0]
  end

  def download_os_human(os)
    if os == "darwin"
      return "Mac OS X"
    elsif os == "freebsd"
      return "FreeBSD"
    elsif os == "openbsd"
      return "OpenBSD"
    elsif os == "Linux"
      return "Linux"
    elsif os == "windows"
      return "Windows"
    else
      return os
    end
  end

  def download_url(file)
    "https://dl.bintray.com/mitchellh/terraform/#{file}"
  end

  def latest_version
    ENV["TERRAFORM_VERSION"]
  end
end
