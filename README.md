# rudeclaude

**Un outil [RudeOps](https://www.rudeops.com), la newsletter DevOps à lire
avant de déployer.** [S'abonner →](https://www.rudeops.com)

Un tableau de bord épuré, en direct dans le terminal, pour suivre son usage de
[Claude Code](https://claude.com/claude-code) : les limites d'usage de
l'abonnement, l'activité du jour et ce que fait Claude dans chaque session.

![rudeclaude en mode image (démo)](docs/capture-image.png)

> **Projet non officiel.** rudeclaude n'est ni affilié à Anthropic, ni
> approuvé par Anthropic. « Claude » et « Claude Code » sont des marques
> d'Anthropic.

## Ce qu'il affiche

- **Limites** : le pourcentage consommé sur la fenêtre de 5 heures et sur la
  semaine, et le temps restant avant chaque remise à zéro. L'anneau passe du
  jaune à l'orange à partir de 70 %, puis au rouge à partir de 90 %. Le petit
  point indique le temps écoulé dans la fenêtre : si l'arc le dépasse, la
  consommation va plus vite que le temps.
- **Aujourd'hui** : le nombre de réponses de Claude, les tokens générés et la
  part des tokens d'entrée servis par le cache.
- **Activité** : les tokens générés minute par minute sur la dernière heure.
- **Sessions** : chaque session active dans l'heure, avec son projet, sa
  branche git, son modèle, la taille de son contexte et ce que fait Claude :
  - vert : il travaille (« exécute une commande », « modifie un fichier »…) ;
  - jaune : il vous pose une question et attend votre réponse ;
  - gris : il a terminé, ou la session est inactive.
- **Outils** les plus utilisés dans l'heure, et **crédits extra** consommés
  dans le mois.

## Prérequis

- **Linux.** macOS n'est pas encore pris en charge : Claude Code y range ses
  identifiants dans le trousseau, que rudeclaude ne sait pas encore lire.
- **Claude Code**, connecté avec un abonnement Claude (Pro ou Max).
- **Go 1.27** ou plus récent, pour compiler.
- Un terminal en couleurs 24 bits. Pour le rendu en images : un terminal qui
  supporte le [protocole graphique de kitty](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
  (Ghostty, kitty, WezTerm, Konsole…).

## Installation

```bash
go install github.com/rudeops/rudeclaude@latest
```

Ou depuis les sources :

```bash
git clone https://github.com/rudeops/rudeclaude.git
cd rudeclaude
go build -o rudeclaude .
```

## Utilisation

```bash
rudeclaude
```

| Option | Effet |
|---|---|
| `--interval 2m` | délai entre deux appels à l'API d'usage (1 min par défaut, 30 s minimum) |
| `--render auto\|image\|text` | force le rendu (détecté automatiquement par défaut) |
| `--demo` | données fictives, sans appel réseau ni lecture des journaux |
| `--snapshot fichier.png` | exporte une image du tableau de bord, puis quitte |
| `--version` | affiche la version, puis quitte |

Touches : `r` pour rafraîchir, `q` pour quitter.

### Rendus

- **Images** : dans les terminaux compatibles, le tableau de bord est dessiné
  en vraies images lissées (police Inter), avec un fond transparent.
- **Texte** : partout ailleurs (y compris dans tmux), la même vue en caractères.

![rudeclaude en mode texte (démo)](docs/capture-texte.png)

## Confidentialité et sécurité

rudeclaude ne lit que deux choses dans `~/.claude` (ou dans
`$CLAUDE_CONFIG_DIR`) :

1. **`.credentials.json`**, pour les limites d'usage. Le jeton OAuth de
   Claude Code y est lu à chaque appel et envoyé **uniquement** à
   `api.anthropic.com`. Il n'est jamais affiché, copié ni rafraîchi : s'il a
   expiré, relancez `claude`.
2. **`projects/**/*.jsonl`**, les journaux de Claude Code, pour l'activité et
   les sessions. Seules les métadonnées sont exploitées : horodatage, projet,
   branche, modèle, compteurs de tokens et noms des outils. Le contenu des
   conversations n'est jamais interprété, stocké ni transmis.

La dernière réponse de l'API d'usage est mise en cache dans
`~/.cache/rudeclaude/usage.json` (sans le jeton), pour que plusieurs fenêtres
ouvertes n'appellent pas l'API plus d'une fois par minute au total.

Attention : l'export `--snapshot` montre les noms de vos projets et de vos
branches.

## Limites connues

- L'API d'usage (`/api/oauth/usage`) n'est **pas documentée** par Anthropic et
  peut changer ou disparaître sans préavis.
- Seule l'activité de Claude Code sur la machine locale est visible. Les
  conversations sur claude.ai ou dans l'application de bureau comptent dans
  les limites, mais pas dans l'activité.
- Une demande d'autorisation en attente (avant de lancer une commande) ne
  laisse pas de trace dans les journaux : la session reste affichée comme
  « exécute une commande », avec la durée écoulée au-delà de 2 minutes.
- L'interface est en français.

## Structure du code

```
main.go                 options et choix du rendu
internal/usage/         appel à l'API d'usage, jeton, cache partagé
internal/activity/      suivi des sessions dans les journaux de Claude Code
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

À l'origine, RudeOps, c'était juste l'envie d'arrêter d'envoyer des liens en
vrac sur Slack et de perdre des ressources intéressantes dans des discussions
qui disparaissent deux jours plus tard. Alors j'en ai fait une newsletter.
Sans plan, sans tunnel de conversion, sans bruit inutile : de la veille tech,
du DevOps, de l'open source, des outils utiles, et parfois quelques
réflexions au passage.

**+2 000 abonnés** · **58,7 %** de taux d'ouverture moyen · depuis 2023, et
toujours vivant.

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
