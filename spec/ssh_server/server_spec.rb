RSpec.describe SSHServer::Server do
  let(:server) { described_class.new }

  describe 'start' do
    let(:port) { 2222 }

    before { server.start(address: 'localhost', port: port, login: 'admin', password: 'password', executable: 'bash --rcfile') }
    after  { server.stop }

    it 'process exist' do
      expect(`ps -p #{server.pid} | grep #{server.pid}`.strip).to include(server.pid.to_s)
    end

    it 'allow to login via ssh' do
      sleep(2)

      Net::SSH.start('localhost', 'admin', { port: port, password: 'password' }) do |ssh|
        puts ssh.exec!('uname -a')
      end
    end
  end
end
