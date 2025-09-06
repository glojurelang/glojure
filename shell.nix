{ sources ? import ./nix/sources.nix
, pkgs ? import sources.nixpkgs {}
}:

# set MACOSX_DEPLOYMENT_TARGET=12.0 env var in shell

pkgs.mkShell {
  nativeBuildInputs = [
    pkgs.go_1_21
    pkgs.clojure
  ];

  shellHook = ''
        export MACOSX_DEPLOYMENT_TARGET=12.0
    '';
}
