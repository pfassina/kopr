#!/bin/sh
# kopr-latex: LaTeX to Unicode converter for render-markdown.nvim
# Reads LaTeX from stdin (without $ delimiters), writes Unicode to stdout.
#
# Uses awk for proper \command tokenisation — each backslash-letter sequence
# is looked up in a symbol table. Unknown commands pass through unchanged.

exec awk '
BEGIN {
  # Greek lowercase
  s["alpha"]="α"; s["beta"]="β"; s["gamma"]="γ"; s["delta"]="δ"; s["epsilon"]="ε"
  s["zeta"]="ζ"; s["eta"]="η"; s["theta"]="θ"; s["iota"]="ι"; s["kappa"]="κ"
  s["lambda"]="λ"; s["mu"]="μ"; s["nu"]="ν"; s["xi"]="ξ"; s["pi"]="π"
  s["rho"]="ρ"; s["sigma"]="σ"; s["tau"]="τ"; s["upsilon"]="υ"; s["phi"]="φ"
  s["chi"]="χ"; s["psi"]="ψ"; s["omega"]="ω"; s["varepsilon"]="ε"; s["varphi"]="φ"

  # Greek uppercase
  s["Gamma"]="Γ"; s["Delta"]="Δ"; s["Theta"]="Θ"; s["Lambda"]="Λ"; s["Xi"]="Ξ"
  s["Pi"]="Π"; s["Sigma"]="Σ"; s["Phi"]="Φ"; s["Psi"]="Ψ"; s["Omega"]="Ω"

  # Operators
  s["times"]="×"; s["div"]="÷"; s["pm"]="±"; s["mp"]="∓"; s["cdot"]="·"
  s["star"]="⋆"; s["circ"]="∘"; s["bullet"]="∙"; s["oplus"]="⊕"; s["otimes"]="⊗"

  # Relations
  s["leq"]="≤"; s["geq"]="≥"; s["neq"]="≠"; s["approx"]="≈"; s["equiv"]="≡"
  s["sim"]="∼"; s["simeq"]="≃"; s["cong"]="≅"; s["propto"]="∝"
  s["ll"]="≪"; s["gg"]="≫"; s["le"]="≤"; s["ge"]="≥"; s["ne"]="≠"

  # Arrows
  s["rightarrow"]="→"; s["leftarrow"]="←"; s["leftrightarrow"]="↔"
  s["Rightarrow"]="⇒"; s["Leftarrow"]="⇐"; s["Leftrightarrow"]="⇔"
  s["uparrow"]="↑"; s["downarrow"]="↓"; s["mapsto"]="↦"; s["to"]="→"
  s["implies"]="⟹"; s["iff"]="⟺"

  # Set theory
  s["in"]="∈"; s["notin"]="∉"; s["subset"]="⊂"; s["supset"]="⊃"
  s["subseteq"]="⊆"; s["supseteq"]="⊇"; s["cup"]="∪"; s["cap"]="∩"
  s["emptyset"]="∅"; s["varnothing"]="∅"

  # Calculus and big operators
  s["int"]="∫"; s["iint"]="∬"; s["iiint"]="∭"; s["oint"]="∮"
  s["sum"]="∑"; s["prod"]="∏"; s["coprod"]="∐"
  s["partial"]="∂"; s["nabla"]="∇"

  # Logic
  s["forall"]="∀"; s["exists"]="∃"; s["nexists"]="∄"; s["neg"]="¬"
  s["land"]="∧"; s["lor"]="∨"; s["lnot"]="¬"

  # Misc
  s["infty"]="∞"; s["infinity"]="∞"
  s["sqrt"]="√"; s["cbrt"]="∛"
  s["angle"]="∠"; s["triangle"]="△"
  s["ldots"]="…"; s["cdots"]="⋯"; s["vdots"]="⋮"; s["ddots"]="⋱"
  s["langle"]="⟨"; s["rangle"]="⟩"
  s["lceil"]="⌈"; s["rceil"]="⌉"; s["lfloor"]="⌊"; s["rfloor"]="⌋"
  s["ell"]="ℓ"; s["hbar"]="ℏ"; s["Re"]="ℜ"; s["Im"]="ℑ"; s["wp"]="℘"
  s["aleph"]="ℵ"

  # Commands to remove silently
  s["left"]=""; s["right"]=""; s["quad"]=" "; s["qquad"]="  "
  s["frac"]=""; s["text"]=""; s["mathrm"]=""; s["mathbf"]=""; s["mathit"]=""

  # Superscript digits
  sup["0"]="⁰"; sup["1"]="¹"; sup["2"]="²"; sup["3"]="³"; sup["4"]="⁴"
  sup["5"]="⁵"; sup["6"]="⁶"; sup["7"]="⁷"; sup["8"]="⁸"; sup["9"]="⁹"
  sup["n"]="ⁿ"; sup["i"]="ⁱ"; sup["+"]="⁺"; sup["-"]="⁻"

  # Subscript digits
  lo["0"]="₀"; lo["1"]="₁"; lo["2"]="₂"; lo["3"]="₃"; lo["4"]="₄"
  lo["5"]="₅"; lo["6"]="₆"; lo["7"]="₇"; lo["8"]="₈"; lo["9"]="₉"
  lo["a"]="ₐ"; lo["e"]="ₑ"; lo["i"]="ᵢ"; lo["n"]="ₙ"; lo["x"]="ₓ"
  lo["+"]="₊"; lo["-"]="₋"
}

# Convert a string using a character map (sup or sub)
function to_script(str, map,    i, c, out) {
  out = ""
  for (i = 1; i <= length(str); i++) {
    c = substr(str, i, 1)
    out = out (c in map ? map[c] : c)
  }
  return out
}

{
  # Phase 1: replace \command sequences via symbol table
  result = ""
  rest = $0
  while (match(rest, /\\[a-zA-Z]+/)) {
    result = result substr(rest, 1, RSTART - 1)
    cmd = substr(rest, RSTART + 1, RLENGTH - 1)
    if (cmd in s)
      result = result s[cmd]
    else
      result = result substr(rest, RSTART, RLENGTH)
    rest = substr(rest, RSTART + RLENGTH)
  }
  result = result rest

  # Phase 2: \frac{a}{b} → a/b (handled after command replacement
  # since \frac was replaced by "" leaving {a}{b})
  while (match(result, /\{[^}]*\}\{[^}]*\}/)) {
    # Only convert if this looks like it was a frac (preceded by nothing special)
    pre = substr(result, 1, RSTART - 1)
    body = substr(result, RSTART, RLENGTH)
    post = substr(result, RSTART + RLENGTH)
    # Extract the two groups
    if (match(body, /^\{([^}]*)\}\{([^}]*)\}$/)) {
      a = substr(body, 2, index(body, "}{") - 2)
      b_start = index(body, "}{") + 2
      b = substr(body, b_start, length(body) - b_start)
      result = pre a "/" b post
    } else {
      break
    }
  }

  # Phase 3: superscripts ^{...} and ^x
  while (match(result, /\^{[^}]*}/)) {
    pre = substr(result, 1, RSTART - 1)
    body = substr(result, RSTART + 2, RLENGTH - 3)
    post = substr(result, RSTART + RLENGTH)
    result = pre to_script(body, sup) post
  }
  while (match(result, /\^[0-9a-zA-Z]/)) {
    pre = substr(result, 1, RSTART - 1)
    c = substr(result, RSTART + 1, 1)
    post = substr(result, RSTART + 2)
    result = pre (c in sup ? sup[c] : "^" c) post
  }

  # Phase 4: subscripts _{...} and _x
  while (match(result, /_{[^}]*}/)) {
    pre = substr(result, 1, RSTART - 1)
    body = substr(result, RSTART + 2, RLENGTH - 3)
    post = substr(result, RSTART + RLENGTH)
    result = pre to_script(body, lo) post
  }
  while (match(result, /_[0-9a-zA-Z]/)) {
    pre = substr(result, 1, RSTART - 1)
    c = substr(result, RSTART + 1, 1)
    post = substr(result, RSTART + 2)
    result = pre (c in lo ? lo[c] : "_" c) post
  }

  # Phase 5: strip remaining braces used for grouping
  gsub(/{/, "", result)
  gsub(/}/, "", result)

  print result
}
'
