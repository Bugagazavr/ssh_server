module SSHServer
  module Wrapper
    extend FFI::Library
    ffi_lib SSHServer.lib_path
    attach_function :start, [:string, :string, :string, :string, :string], :void
  end
end
