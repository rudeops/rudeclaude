# rudeclaude

**Français** · [English](README.en.md)

**Un outil propulsé par [RudeOps](https://www.rudeops.com), la newsletter de
veille sur la tech en général et le DevOps en particulier.**

![rudeclaude en mode image (démo)](docs/capture-image.png)

**Claude Code travaille. rudeclaude compte.**

Un dashboard minimaliste en TUI, pour savoir où j'en suis de mon abonnement
avant que Claude Code ne me l'annonce lui-même en plein milieu d'un
refacto. Limites d'usage, activité du jour, et ce que fait Claude
dans chaque session, pendant que je regarde ailleurs.

> **Projet non officiel.** rudeclaude n'est ni affilié à Anthropic, ni
> approuvé par Anthropic. "Claude" et "Claude Code" sont des marques
> d'Anthropic.

## Ce qu'il affiche

- **Limites** : le pourcentage consommé sur la fenêtre de 5 heures et sur la
  semaine, et le temps restant avant chaque remise à zéro. L'anneau passe du
  jaune à l'orange à partir de 70 %, puis au rouge à partir de 90 %. Le petit
  point indique le temps écoulé dans la fenêtre : si l'arc le dépasse, je
  consomme plus vite que le temps ne passe. Mauvais signe.
- **Aujourd'hui** : le nombre de réponses de Claude, les tokens générés et la
  part des tokens d'entrée servis par le cache.
- **Semaine par produit** : la part de la semaine partie dans Claude Code, dans
  le chat et dans Cowork. Pour savoir enfin qui a mangé le quota.
- **Activité** : les tokens générés minute par minute sur la dernière heure.
  Pratique pour retrouver le moment exact où tout a dérapé.
- **RTK** : si [rtk](https://github.com/rtk-ai/rtk) est installé, les tokens
  qu'il a économisés aujourd'hui et au total, son taux moyen et un histogramme
  des 7 derniers jours. Sans rtk, le bloc disparaît et l'activité prend toute
  la largeur. Aucune pression.
- **Sessions** : chaque session active dans l'heure, avec son projet, sa
  branche git, son modèle, la taille de son contexte et ce que fait Claude :
  - vert : il travaille ("exécute une commande", "modifie un fichier"…) ;
  - jaune : il vous a posé une question et attend votre réponse. Il est
    patient, lui ;
  - gris : il a terminé, ou la session dort.
- **Outils** les plus utilisés dans l'heure, et **crédits extra** consommés
  dans le mois. Celui-là, je le regarde entre mes doigts.

## Prérequis

- **Linux.** macOS n'est pas encore pris en charge : Claude Code y range ses
  identifiants dans le trousseau, et rudeclaude ne sait pas encore l'ouvrir.
- **Claude Code**, connecté avec un abonnement Claude (Pro ou Max).
- **Go 1.27** ou plus récent, pour compiler.
- Un terminal en couleurs 24 bits. Pour le rendu en images, un terminal qui
  parle le [protocole graphique de kitty](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
  (Ghostty, kitty, WezTerm, Konsole…).
- En option, [rtk](https://github.com/rtk-ai/rtk) dans le `PATH`, pour le bloc
  RTK.

## Installation

```sh
go install github.com/rudeops/rudeclaude@latest
```

Ou depuis les sources :

```sh
git clone https://github.com/rudeops/rudeclaude.git
cd rudeclaude
go build -o rudeclaude .
```

## Utilisation

```sh
rudeclaude
```

C'est tout. Enfin, presque :

| Option | Effet |
|---|---|
| `--interval 2m` | délai entre deux appels à l'API d'usage (1 min par défaut, 30 s minimum) |
| `--render auto\|image\|text` | force le rendu (détecté automatiquement par défaut) |
| `--demo` | données fictives, sans appel réseau ni lecture des journaux |
| `--snapshot fichier.png` | exporte une image du tableau de bord, puis quitte |
| `--version` | affiche la version, puis quitte |

Touches : `r` pour rafraîchir, `q` pour quitter. Il n'y en a pas de troisième.

### Rendus

- **Images** : dans les terminaux compatibles, le tableau de bord est dessiné
  en vraies images lissées (police Inter), sur fond transparent. Oui, dans un
  terminal.
- **Texte** : partout ailleurs, y compris dans tmux, la même vue en
  caractères. Même vérité, moins de pixels.

![rudeclaude en mode texte (démo)](docs/capture-texte.png)

## Confidentialité et sécurité

Je n'aime pas les outils qui fouillent dans mes dossiers sans prévenir. Voilà
donc exactement ce que rudeclaude lit dans `~/.claude` (ou dans
`$CLAUDE_CONFIG_DIR`), et rien d'autre :

1. **`.credentials.json`**, pour les limites d'usage. Le jeton OAuth de
   Claude Code y est lu à chaque appel et envoyé **uniquement** à
   `api.anthropic.com`. Il n'est jamais affiché, copié ni rafraîchi : s'il a
   expiré, relancez `claude`.
2. **`projects/**/*.jsonl`**, les journaux de Claude Code, pour l'activité et
   les sessions. Seules les métadonnées sont exploitées : horodatage, projet,
   branche, modèle, compteurs de tokens et noms des outils. Le contenu des
   conversations n'est jamais interprété, stocké ni transmis. Vos prompts ne
   l'intéressent pas.

Si rtk est installé, rudeclaude lance aussi `rtk gain --daily --format json`
toutes les 30 secondes. Ses chiffres restent sur la machine.

La dernière réponse de l'API d'usage est mise en cache dans
`~/.cache/rudeclaude/usage.json` (sans le jeton), pour que plusieurs fenêtres
ouvertes n'appellent pas l'API plus d'une fois par minute au total. Ouvrez-en
dix si ça vous rassure.

Attention : l'export `--snapshot` montre les noms de vos projets et de vos
branches. Relisez avant de partager une capture où traîne
`hotfix/montargis-panique`.

## Limites connues

- L'API d'usage (`/api/oauth/usage`) n'est **pas documentée** par Anthropic et
  peut changer ou disparaître sans préavis. Si rudeclaude casse un matin sans
  raison, c'est probablement elle.
- Seule l'activité de Claude Code sur la machine locale est visible. Les
  conversations sur claude.ai ou dans l'application de bureau comptent dans
  les limites (et dans la semaine par produit), mais pas dans l'activité. Si
  le pourcentage grimpe alors que l'activité ne bouge pas, c'est vous, sur
  votre téléphone.
- Une demande d'autorisation en attente (avant de lancer une commande) ne
  laisse aucune trace dans les journaux : la session reste affichée comme
  "exécute une commande", avec la durée écoulée au-delà de 2 minutes. Si ça
  s'éternise, allez voir : il attend sûrement votre feu vert.
- rtk découpe ses journées en UTC : juste après minuit, son "aujourd'hui"
  est encore un peu hier.
- L'interface est en français. Je ne compte pas m'en excuser, mais d'autres
  langues pourraient arriver dans de futures versions.

## Structure du code

```
main.go                 options et choix du rendu
internal/usage/         appel à l'API d'usage, jeton, cache partagé
internal/activity/      suivi des sessions dans les journaux de Claude Code
internal/rtk/           économies de tokens de rtk, s'il est installé
internal/gfx/           dessin en images de la vue d'ensemble, police Inter
internal/kitty/         protocole graphique de kitty
internal/ui/            boucle du mode image, repli texte (Bubble Tea)
internal/theme/         couleurs du mode texte
```

Les tests se lancent avec `go test ./...`.

## La newsletter RudeOps

rudeclaude est né dans l'atelier de [RudeOps](https://www.rudeops.com), une
newsletter de veille tech et DevOps indépendante, pour celles et ceux qui font
tourner des systèmes en production.

Sur le site, en plus de la newsletter : des [guides](https://www.rudeops.com/guides)
(le DevOps, le modèle CAMS, les trois chemins…), un
[glossaire de la tech](https://www.rudeops.com/rudefinitions) et des quiz pour
préparer les certifications Kubernetes KCNA et KCSA.

**→ [S'abonner sur rudeops.com](https://www.rudeops.com)** ·
[Archives](https://www.rudeops.com/newsletters) ·
[RSS](https://www.rudeops.com/feed.xml)

## Licence

[GPL v3](LICENSE). La police [Inter](https://rsms.me/inter/) est incluse sous
licence [OFL](internal/gfx/fonts/LICENSE.txt).
