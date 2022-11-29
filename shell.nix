{ sources ? import ./nix/sources.nix
, pkgs ? import sources.nixpkgs {}
}:

pkgs.mkShell {
  nativeBuildInputs = [
    pkgs.go_1_19
    pkgs.clojure
  ];
}
