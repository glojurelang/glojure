{ sources ? import ./nix/sources.nix
, pkgs ? import sources.nixpkgs {}
}:

# set MACOSX_DEPLOYMENT_TARGET=12.0 env var in shell

pkgs.mkShell {
  nativeBuildInputs = [
    pkgs.go_1_24
    pkgs.clojure
  ];

  shellHook = ''
        export MACOSX_DEPLOYMENT_TARGET=12.0

        # Remove xcbuild's xcrun from PATH
        export PATH=$(echo "$PATH" | sed "s|${pkgs.xcbuild.xcrun}/bin||g")

        # Let Xcode pick its own developer dir
        unset DEVELOPER_DIR
    '';
}
