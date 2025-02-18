{ pkgs, buildGoModule, arch ? "x86" }:
with pkgs; buildGoModule rec {
  name = "security-profiles-operator";
  src = builtins.filterSource
    (path: type: !(type == "directory" && baseNameOf path == "build")) ./..;
  vendorSha256 = null;
  doCheck = false;
  outputs = [ "out" ];
  nativeBuildInputs = with buildPackages; [
    bpftool
    git
    llvmPackages_13.clang-unwrapped
    llvm_13
    pkg-config
    which
  ];
  buildInputs = [
    (libseccomp.overrideAttrs (x: { dontDisableStatic = true; }))
    glibc
    glibc.static
    libbpf
    libelf
    zlib.static
  ];
  buildPhase = ''
    export CFLAGS=$NIX_CFLAGS_COMPILE
    export ARCH=${arch}
    make build/recorder.bpf.o
  '';
  installPhase = ''
    install -Dm644 -t $out build/recorder.bpf.o
  '';
}
