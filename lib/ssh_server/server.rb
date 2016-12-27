module SSHServer
  class Server
    attr_reader :pid

    def intialize
      @pid = nil
    end

    def start(options = {})
      options[:address]    ||= '0.0.0.0'
      options[:port]       ||= 2222
      options[:login]      ||= 'admin'
      options[:password]   ||= 'password'
      options[:executable] ||= %w(bash)

      %w(TERM INT).each do |s|
        Signal.trap(s) do
          Thread.new do
            stop
          end
        end
      end

      server(options)
    end

    def stop
      if pid
        Process.kill('KILL', pid)
      end
    end

    def server(options = {})
      @pid = Process.fork do
        require 'ssh_server/wrapper'

        Wrapper.start(
          "#{options[:address]}:#{options[:port]}",
          options[:login],
          options[:password],
          options[:executable],
          options[:ssh_key] || generate_key
        )
      end
    end

    private

    def generate_key
      @generate_key ||= begin
                          file = Tempfile.new('pkey')
                          file.write(OpenSSL::PKey::RSA.generate(2048).to_s)
                          file.close
                          file.path
                        end
    end
  end
end
