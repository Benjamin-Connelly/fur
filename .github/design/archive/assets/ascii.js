// ───────────────────────────────────────────────────────────────────
// ASCII companions — v3, drawn in the 2ch / Joan Stark / Samamine
// tradition. The character ∧ (U+2227) is used for ears — it's the
// canonical Japanese ASCII cat ear glyph, and it reads pointier than
// a literal /\. Kaomoji faces (=^·ω·^=), (=ΦωΦ=), (´∀｀) are the
// grammar of cat-ness in text. Blocks are used sparingly, for
// deliberate silhouettes only (02, 04, 11).
// ───────────────────────────────────────────────────────────────────
window.ASCII = {
  // 01 — Scholarly cat with terminal-bracket reading glasses.
  //      Kaomoji body + explicit [_] bracket lenses + >_ prompt.
  "01": String.raw`
      ∧_____∧
     (         )
     | [_]-[_] |
     |    ·    |
     |   >_    |
      \_______/
        U U`,

  // 02 — Chat Noir. Solid silhouette. Blocks are earned here.
  //      Brass ◆ eyes, classical dangling tail.
  "02": String.raw`
     ▄██▄  ▄██▄
    ████████████
    ████◆  ◆████
    █████████████
    █████████████
     ███████████
       ██   ██
         ╲
          ╲
           ╲`,

  // 03 — Harper Modular. Overlapping geometric shapes, flat line-art.
  //      Minimal vocabulary: ∧ / \ | _ ● ω
  "03": String.raw`
      ∧∧        ∧∧
     /  \      /  \
    |    \____/    |
    |  ●        ●  |
     \     ω      /
      \__________/
         U    U`,

  // 04 — NES sprite. Pure block grid.
  "04": String.raw`
    ██      ██
   ████    ████
  ██████████████
  ██  ██████  ██
  ██  ██████  ██
  ██████████████
  ████▓▓▓▓████
  ██████████████
    ██████████
      ██████`,

  // 05 — Monocle scholar. Asymmetric: (O) + chain beads.
  "05": String.raw`
       ∧___∧
      ( |  (O)═╗
       \_____/ ║
               ●
               ∙
               ∙
               ○`,

  // 06 — Cat in terminal window. Classic kaomoji + prompt.
  //      The (")_(") "paws on floor" tradition.
  "06": String.raw`
  ┌ ● ● ○ ─── fur ──┐
  │ ~/docs/          │
  │                  │
  │      ∧___∧       │
  │     (=^·.·^=)    │
  │     (")_(")      │
  │                  │
  │ $ fur README.md ▮│
  └──────────────────┘`,

  // 07 — Minimal cat, tail becomes cursor. Kaomoji face + hanging block.
  "07": String.raw`
      ∧___∧
     (= •ω• =)
      )    (            ▌
       \    \           ▌
        \    \_________▌
         \            ▌
                      ▀`,

  // 08 — Wain hypnotic bullseye. Concentric ○◎● + symmetric whiskers.
  "08": String.raw`
      ∧∧          ∧∧
    ∧∧∧∧        ∧∧∧∧
   ╭──────────────────╮
   │ ●○◎○●    ●○◎○● │
   │                   │
   │         ω         │
   │        \_/        │
   │      ~~~~~        │
    ╲_________________╱`,

  // 09 — Calico tricolor patches. ▓ ▒ ░ as color zones in a framed face.
  "09": String.raw`
     ▓▓∧___∧░░
    ▓▓(  ●ω●  )░░
     ▓▓\  ·  /░░
       \____/
        U U`,

  // 10 — Scaredy cat, Japanese 威嚇 (intimidation) pose.
  //      Spikes radiate from arched posture.
  "10": String.raw`
     \  \ | | /  /
    \\\  ∧∧∧∧  ///
    ==( O  O )==
       \ ω /
       / | \
      /  |  \
       ! !`,

  // 11 — Vintage Halloween. Spiky + fanged. Half-block for mouth only.
  "11": String.raw`
   ∧∧∧∧∧∧∧∧∧∧∧
  ∧∧∧∧∧∧∧∧∧∧∧∧∧
   (Ó)       (Ó)
        ___
       \▼▼▼/
        \_/
   ∨∨∨∨∨∨∨∨∨∨∨`,

  // 12 — Tattoo flash. Bold outlined with banner tails.
  "12": String.raw`
    ★    ∧___∧    ★
        ( ●_● )
         \ ♥ /
          \_/
     ──────────────
     ❯  F · U · R  ❮`,

  // 13 — Low-poly triangulated. Pure triangle vocabulary.
  "13": String.raw`
        ∧∧          ∧∧
      ∧▽▼∧        ∧▼▽∧
     ▲▽▼▲▽▼▲▼▽▼▲▽▼▲
      ▼ ◉ ▲ ▼ ◉ ▽
       ▽▲▽▲▽▲▽
        ▽▼▽▼▽
          ▽▽`,

  // 14 — ★ Scholar's Stack (the lead).
  //      Kaomoji cat with brass glasses sitting on a labeled book stack.
  "14": String.raw`
         ∧_____∧
        ( ◉ _ ◉ )
         \_____/
           U U
     ╔═══════════════╗
     ║  K&R · C      ║
     ╠═══════════════╣
     ║  SICP         ║
     ╠═══════════════╣
     ║  cat(1)       ║
     ╠═══════════════╣
     ║  fur(1) ← new ║
     ╚═══════════════╝`,

  // 15 — Kuroneko badge. Walking-cat silhouette in a rounded rect.
  "15": String.raw`
  ╭───────────────────╮
  │                   │
  │  ∧    ∧∧∧∧∧∧      │
  │ ▐█▄__████████▄    │
  │   ▘  █▘   █▘  ▘   │
  │                   │
  ╰───────────────────╯`,

  // 16 — Ink noir. Sparse drippy silhouette with diamond oxblood eyes.
  "16": String.raw`
      ·              ·
           ∧___∧
         ░(◆ ω ◆)░
           /|_|\
            vvv
       │  │  │  │  │
       ▌  ▌  ▌  ▌  ▌
            drip`
};
