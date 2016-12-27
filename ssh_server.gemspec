$:.push File.expand_path("../lib", __FILE__)
require 'ssh_server/version'

Gem::Specification.new do |s|
  s.name        = "ssh_server"
  s.version     = SSHServer::VERSION
  s.summary     = "Standalone ssh server writen on go"
  s.description = "Standalone ssh server writen on go"

  s.license = "MIT"

  s.author   = "Kirill Zaitsev"
  s.email    = "kirik910@gmail.com"

  s.files        = Dir["lib/**/*", "ext/**/*"]
  s.require_path = "lib"

  s.add_dependency "ffi"

  s.add_development_dependency 'rspec', '~> 3.5.0'
  s.add_development_dependency 'pry'
  s.add_development_dependency 'net-ssh'
end
