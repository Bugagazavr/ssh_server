require 'ffi'
require 'singleton'

module SSHServer
  class << self
    def lib_path
      @lib_path ||= File.expand_path(File.join('..', '..', 'ext', lib_file), __FILE__)
    end

    def lib_file
      @lib_file ||= case `uname -s`.strip.downcase
                    when 'darwin'
                      "ssh_server-darwin-10.6-#{arch}.dylib"
                    when 'linux'
                      "ssh_server-linux-#{arch}.so"
                    else
                      raise 'Unsupported OS'
                    end
    end

    private

    def arch
      @arch ||= case `uname -m`.strip.downcase
                when 'i386'
                  '386'
                when 'x86_64', 'amd64'
                  'amd64'
                else
                  raise 'Unsupported architecture'
                end
    end
  end
end

require 'ssh_server/server'
require 'ssh_server/version'
