/* jigger — bascule de langue (fichier externe : compatible CSP 'self'). */
(function () {
  var docEl = document.documentElement;

  /* --- FR --- */
  var FR = {
    /* --- les titres de page : une clé par page, posée sur son <title> --- */
    'title.home': 'jigger — une syntaxe pour Homebrew, winget, scoop et pacman',
    'title.use': 'jigger — télécharger, brancher, utiliser',
    'title.ssh': 'jigger — le sélecteur SSH',
    /* --- partagé : en-tête et pied --- */
    'nav.home': 'Accueil',
    'nav.use': 'L’utiliser',
    'nav.ssh': 'SSH',
    /* --- page « utiliser » --- */
    'use.eyebrow': 'L’utiliser',
    'use.h1': 'Télécharger, brancher, utiliser.',
    'use.showing': 'Commandes affichées pour',
    'use.dl.h2': 'Le télécharger.',
    'use.dl.mac': 'Homebrew, depuis le tap du projet — hébergé sur le GitLab du projet, d’où l’URL explicite. La formule construit le binaire sur votre machine, en tirant <span class="mono">go</span> comme dépendance de compilation, et installe le greffon zsh à côté.',
    'use.dl.macup': 'Pour mettre à jour ensuite : <span class="mono">brew upgrade jigger</span>.',
    'use.dl.win': 'scoop, depuis le bucket du projet. Depuis la v0.10.0, les publications embarquent des binaires précompilés : rien à compiler, pas de Go à installer. <span class="mono">scoop bucket add</span> prend deux arguments — le nom local que vous choisissez, puis le dépôt ; ne passer que le nom fait chercher scoop dans son propre annuaire, et il répond <span class="mono">unknown bucket</span>.',
    'use.dl.winup': 'Pour mettre à jour ensuite : <span class="mono">scoop update jigger</span>. Il n’existe pas de paquet winget.',
    'use.dl.omarchy': 'Go, parce qu’il n’existe <strong>aucun paquet jigger</strong> — ni dans les dépôts d’Arch, ni dans l’AUR. Ce n’est pas une route au rabais : jigger n’a aucune dépendance d’exécution en dehors des gestionnaires eux-mêmes, si bien que le binaire compilé est tout ce qu’il y a.',
    'use.dl.omarchypath': 'Le binaire arrive dans <span class="mono">$GOBIN</span> — <span class="mono">~/go/bin</span> par défaut. Le mettre sur le <span class="mono">PATH</span> s’il n’y est pas :',
    'use.dl.omarchyup': 'Pour mettre à jour ensuite : la même ligne <span class="mono">go install</span> — <span class="mono">@latest</span> va chercher le dernier tag à chaque fois.',
    'use.dl.go': 'Toute plateforme, si vous avez Go : <span class="mono">go install gitlab.yg-devworks.com/yves/jigger@latest</span>',
    'use.wire.h2': 'Le brancher dans votre shell.',
    'use.wire.mac': 'Une ligne <span class="mono">source</span> dans <span class="mono">~/.zshrc</span>, puis <span class="mono">exec zsh</span>. Elle couvre tous les gestionnaires présents sur la machine : le greffon n’a pas besoin qu’on lui dise s’il a affaire à brew ou à pacman, il regarde.',
    'use.wire.macorder': 'L’ordre des appels <span class="mono">source</span> n’a pas d’importance, starship compris — le greffon se place là où il doit dans les hooks de zsh.',
    'use.wire.win': 'Le bucket n’a installé que le <strong>binaire</strong>. Le module — ce qui dessine la fenêtre — vient du dépôt : il faut donc le cloner d’abord. Le chemin ci-dessous est celui que reprend la suite de cette section ; n’importe quel autre convient, pourvu que la ligne suivante le désigne.',
    'use.wire.winmod': 'Puis une ligne <span class="mono">Import-Module</span> dans <span class="mono">$PROFILE</span> — <span class="mono">notepad $PROFILE</span> l’ouvre, et <span class="mono">New-Item -ItemType File -Path $PROFILE -Force</span> le crée s’il n’existe pas encore — puis <span class="mono">. $PROFILE</span>, ou un nouvel onglet.',
    'use.wire.winorder': 'Ici l’ordre compte pour de bon : si vous utilisez oh-my-posh ou starship, importez jigger <strong>après</strong> lui. Et PSReadLine ne garde que le dernier gestionnaire lié à un raccourci — un profil qui lie <kbd>^R</kbd> après l’import reprend la touche, et la bascule regex devient inaccessible. Le guide montre comment la lui repasser.',
    'use.wire.omarchy': 'Le greffon n’est <strong>pas</strong> dans le module Go — il vient du dépôt, qu’il faut donc cloner d’abord. Le chemin ci-dessous est celui que vise la ligne suivante ; n’importe quel autre convient.',
    'use.wire.omarchysource': 'Puis une ligne <span class="mono">source</span> dans <span class="mono">~/.zshrc</span>, et <span class="mono">exec zsh</span>. Elle couvre les deux gestionnaires d’un coup : le greffon n’a pas besoin qu’on lui dise s’il fait face à pacman ou à yay, il regarde.',
    'use.wire.omarchykeep': 'Garder le clone : <span class="mono">git pull</span> met le greffon à jour, <span class="mono">go install …@latest</span> met le binaire, et les deux voyagent ensemble — le greffon refuse de se charger contre un binaire antérieur à 0.11.0, et le dit.',
    'use.check.h2': 'Vérifier que c’est pris.',
    'use.check.mac': 'La version d’abord, puis qui répond. Un vieux binaire qui masque une installation plus récente dans le <span class="mono">PATH</span> est la panne la plus pénible à diagnostiquer — une ligne tranche.',
    'use.check.win': 'La version d’abord, puis qui répond. Un vieux binaire qui masque une installation plus récente dans le <span class="mono">PATH</span> est la panne la plus pénible à diagnostiquer — une ligne tranche.',
    'use.check.omarchy': 'La version d’abord, puis qui répond. Un vieux binaire qui masque une installation plus récente dans le <span class="mono">PATH</span> est la panne la plus pénible à diagnostiquer — une ligne tranche.',
    'use.check.try': 'Ouvrez ensuite un shell neuf et tapez <span class="mono">brew ins</span> — <span class="mono">pacman ins</span> sur Arch, <span class="mono">winget ins</span> sous Windows — <strong>sans appuyer sur Entrée</strong>. Le cadre doit apparaître sous l’invite et se resserrer à chaque lettre.',
    'use.check.note': 'Rien n’apparaît ? Le greffon le dit quand il refuse de se charger : un message, au démarrage du shell, signale que le binaire est absent du <span class="mono">PATH</span>, ou qu’il est trop ancien pour ce greffon. À la toute première utilisation, le cadre peut afficher « catalogue en préparation… » — jigger ne retient jamais une frappe en attendant un gestionnaire de paquets, il construit son catalogue en tâche de fond.',
    'use.now.h2': 'Tapez une commande.',
    'use.now.lede': 'Il n’y a rien d’autre à apprendre. La fenêtre vit toute seule, et les enregistrements ci-dessous sont ce qu’elle fait vraiment — pas une maquette.',
    'use.now.nativeh': 'Votre gestionnaire, pendant que vous tapez',
    'use.now.native': 'Tapez une commande de gestionnaire : le cadre apparaît seul et se resserre à chaque lettre. Rien à déclencher, rien à retenir.',
    'use.demo.mac01': 'brew install fire — la liste se resserre à chaque lettre, sans rien presser. Pris sur macOS.',
    'use.demo.win01': 'winget install fire — le même cadre et les mêmes touches, un autre catalogue : winget répond avec des identifiants Publisher.Package là où brew affiche des noms de formules nus. Pris sur Windows.',
    'use.demo.omarchy01': '<span class="mono">yay -S visual-studio</span> montre les dépôts et l’AUR dans une seule liste — <span class="mono">◆</span> pour un paquet des dépôts, <span class="mono">▣</span> pour un paquet de l’AUR, <span class="mono">●</span> pour ce qui est déjà installé. Aucun enregistrement n’en existe encore.',
    'use.now.jgh': 'Une syntaxe : <span class="mono">jg</span>',
    'use.now.jg': '<span class="mono">jg install fd</span> atteint celui des gestionnaires qui connaît <span class="mono">fd</span>, sans que vous ayez à savoir lequel. La façade ne fait qu’ajouter : <span class="mono">brew install fd</span> continue de marcher exactement comme avant, fenêtre comprise.',
    'use.demo.mac02': 'jg install fd — la façade répond pour le gestionnaire qui connaît le paquet. Pris sur macOS.',
    'use.demo.win02': 'jg install node — avec deux gestionnaires installés côte à côte, la colonne de droite nomme qui répond pour chaque candidat. Pris sur Windows.',
    'use.demo.omarchy02': 'La façade se comporte pareil, avec un détail qui appartient à Arch : pacman et yay sont deux portes sur la même base, aussi <span class="mono">jg</span> liste-t-il vos paquets <strong>une</strong> fois, jamais deux. Aucun enregistrement n’en existe encore.',
    'use.now.regexh': 'Regex, sur la même ligne',
    'use.now.regex': '<kbd>^R</kbd> bascule le filtre, et rien que le filtre : la ligne, le cadre et les touches ne bougent pas. Le titre indique le mode en cours, et hors de la fenêtre la touche reste la recherche inverse d’historique de votre shell.',
    'use.now.regexhow': 'Quatre choses que la bascule ne dit pas d’elle-même. Le motif n’est <strong>pas ancré</strong> : il correspond n’importe où dans un nom, si bien que basculer élargit souvent la liste avant qu’on la resserre. La casse est ignorée dans les deux modes — basculer ne la change jamais en douce. Cela ne vaut que pour les <strong>noms de paquets</strong> : les verbes, les sous-commandes et les options gardent le filtre par préfixe, où une expression rationnelle n’apprendrait rien. Et un motif qui ne compile pas ne retient <strong>rien</strong> — le cadre le dit, plutôt que de déverser le catalogue entier parce qu’il manque une parenthèse.',
    'use.demo.mac04': 'brew install fire liste les noms qui commencent par fire. ^R passe en regex et arrayfire les rejoint — le motif n’est pas ancré, la liste s’élargit donc avant de se resserrer. Puis (bird|fly) n’en garde que quatre. Pris sur macOS.',
    'use.regex.omarchy': 'La même bascule <kbd>^R</kbd> fonctionne sur Arch, sur les dépôts et l’AUR à la fois. Sa capture n’est pas encore enregistrée — le tape qui la joue est écrit, il reste à le lancer sur une machine joignable.',
    'use.demo.win04': 'winget install fire propose vingt et un candidats par préfixe. ^R, puis (bird|blade), en garde quatre — une alternative qu’aucune recherche par préfixe ne sait exprimer. Pris sur Windows.',
    'use.now.runh': 'Compléter, puis exécuter',
    'use.now.run': '<kbd>⏎</kbd> complète la dernière partie <strong>et</strong> lance la ligne, d’une seule frappe. À partir de là, jigger ne fait plus rien du tout : ce qui défile est la sortie du gestionnaire, relayée telle quelle — barres de progression, invites et élévation comprises.',
    'use.install.mac': 'L’installation par la façade se comporte pareil sur macOS, avec brew qui répond — aucune capture n’en est encore enregistrée. Basculez sur Windows ci-dessus pour en voir une.',
    'use.install.omarchy': 'Pareil sur Arch, avec yay qui répond — et ses propres questions relayées telles quelles, celles de l’AUR comprises, puisque jigger ne fait plus rien du tout une fois la ligne lancée. Aucune capture n’en est encore enregistrée.',
    'use.demo.win05': 'jg install hexy, complété en hexyl, puis lancé — la sortie de scoop, relayée sans y toucher. Pris sur Windows, contre un vrai gestionnaire.',
    'use.now.upgh': 'Et une mise à jour',
    'use.now.upg': 'Même geste, autre verbe. Rien ici n’est propre à l’installation : <span class="mono">upgrade</span>, <span class="mono">search</span> et <span class="mono">outdated</span> se complètent et se lancent de la même façon.',
    'use.upgrade.mac': 'C’est pareil sur macOS, où c’est brew qui met à jour — sa capture n’est pas encore enregistrée. Basculez sur Windows ci-dessus pour en voir une.',
    'use.upgrade.omarchy': 'Même geste sur Arch. À noter que yay pilote et que pacman ne fait que lire, ce qui explique que <span class="mono">jg install --pm pacman</span> n’existe pas tant que yay est installé. Sa capture n’est pas encore enregistrée.',
    'use.demo.win06': 'jg upgrade hyperf, complété en hyperfine, puis lancé — scoop remplace la 1.16.1 par la 1.20.0. Pris sur Windows, contre un vrai gestionnaire.',
    'use.keys.h2': 'Les touches.',
    'use.keys.lede': 'Le même jeu partout — un cadre, un jeu de touches, que ce soit Homebrew, winget, scoop, pacman ou yay qui réponde.',
    'use.keys.k1': 'insère le candidat courant',
    'use.keys.k2': 'complète la dernière partie et lance la ligne, d’une seule frappe',
    'use.keys.k3': 'entre dans la liste, puis descend d’un candidat',
    'use.keys.k4': 'remonte ; sur le premier candidat, rend le clavier au shell',
    'use.keys.k5': 'la même chose, pour qui les préfère aux flèches',
    'use.keys.k6': 'referme la fenêtre pour la ligne en cours — ⇥ la rouvre',
    'use.keys.k7': 'bascule le filtre entre texte brut et regex ; le titre du cadre affiche [regex] tant que c’est actif',
    'use.keys.arrows': '<strong>Vos flèches ne sont jamais confisquées.</strong> Tant que la fenêtre n’a pas le clavier, <kbd>↑</kbd> et <kbd>↓</kbd> restent l’historique du shell. Le cadre dit lequel des deux : la ligne courante soulignée et le pied qui affiche <span class="mono">↑↓ naviguer</span> quand il a le clavier, <span class="mono">↓ parcourir</span> quand il ne l’a pas.',
    'use.keys.fix': 'jigger corrige ce qu’il insère chaque fois que la commande serait fausse autrement : <span class="mono">--cask</span> devant un cask Homebrew, le nom qualifié <span class="mono">main/flux</span> pour un paquet scoop présent dans plusieurs buckets, des guillemets autour d’un identifiant winget qui contient des espaces.',
    'use.set.h2': 'Réglages.',
    'use.set.lede': 'Les réglages sont des variables d’environnement, à poser <strong>avant</strong> le <span class="mono">source</span> ou l’<span class="mono">Import-Module</span> : le greffon lit ses clés et pose ses hooks au chargement. Quatre des douze, celles qui servent dès le premier jour :',
    'use.set.commands': 'Les commandes qui ouvrent la fenêtre — <span class="mono">brew pacman yay ssh scp sftp</span> sous zsh, <span class="mono">winget,scoop,ssh,scp,sftp</span> sous PowerShell. C’est aussi par là qu’on désactive le sélecteur SSH. <span class="mono">jigger</span> et <span class="mono">jg</span> sont toujours ajoutés à ce que vous posez.',
    'use.set.lang': 'Les messages : <span class="mono">en</span> ou <span class="mono">fr</span>. Lue avant <span class="mono">LC_ALL</span>, <span class="mono">LC_MESSAGES</span> et <span class="mono">LANG</span> — c’est comme ça qu’on retrouve le français dans un shell anglophone.',
    'use.set.live': '<span class="mono">1</span> par défaut. <span class="mono">0</span> : plus rien n’apparaît sans qu’on le demande, et <kbd>⇥</kbd> ouvre le sélecteur plein écran. C’est aussi la façon d’isoler un conflit avec un autre greffon d’édition de ligne.',
    'use.set.cache': 'Où sont mis en cache les catalogues — <span class="mono">~/Library/Caches/jigger</span> sur macOS, <span class="mono">%LOCALAPPDATA%\\jigger</span> sur Windows. <span class="mono">jigger prompt --path</span> affiche le fichier réellement utilisé.',
    'use.set.config': 'Les variables d’environnement disparaissent avec le shell. <span class="mono">jigger config</span> ouvre un écran qui les écrit, indique d’où vient chaque valeur — défaut, fichier ou environnement — et ne touche jamais à votre <span class="mono">~/.zshrc</span> ni à votre <span class="mono">$PROFILE</span>.',
    'use.linux.h2': 'Et Linux ?',
    'use.linux.p': 'Arch et Omarchy fonctionnent déjà — pacman et yay, même fenêtre, mêmes verbes. Choisissez <strong>Omarchy</strong> dans l’en-tête et cette page bascule sur leurs commandes de bout en bout. Ce qui manque encore, ce sont les enregistrements, pas les instructions. Le guide d’installation va plus loin.',
    'use.linux.link': 'Lire le guide d’installation <span class="arw">→</span>',
    /* --- page « ssh » --- */
    'ssh.eyebrow': 'Le sélecteur SSH',
    'ssh.h1': 'Pas seulement les gestionnaires de paquets.',
    'ssh.why.eyebrow': 'Pourquoi une page à part',
    'ssh.why.h2': 'Le contrat de complétion n’est pas réservé aux gestionnaires de paquets.',
    'ssh.why.lede': '<span class="mono">ssh</span> n’est pas un gestionnaire de paquets, et c’est justement ce qu’il démontre. C’est la même fenêtre, les mêmes touches et le même cadre que pour <span class="mono">brew</span> ou <span class="mono">winget</span> — seul le catalogue change. Rien dans la fenêtre ne sait qu’elle regarde des serveurs plutôt que des paquets, parce que rien dans le contrat qu’honore un fournisseur ne parle jamais de paquets.',
    'ssh.why.adr': 'C’était écrit avant que le sélecteur existe : un fournisseur annonce des candidats, la fenêtre les dessine. Le sélecteur SSH est ce qui prouve que la règle tient hors de Homebrew, winget, scoop, pacman et yay — et ce à quoi ressemblerait le fournisseur suivant, quoi qu’il complète.',
    'ssh.why.link': 'Lire l’ADR-0005 <span class="arw">→</span>',
    'ssh.action.eyebrow': 'En action',
    'ssh.action.h2': 'Tapez <span class="mono">ssh</span> et un espace.',
    'ssh.action.lede': 'Les enregistrements ci-dessous lisent le même <span class="mono">~/.ssh/config</span> — un fixture de serveurs inventés, identique d’un système à l’autre. Rien, dans SSH, n’est propre à une plateforme, et c’est pourquoi un seul enregistrement suffit à montrer ce qu’ils font tous.',
    'ssh.demo.mac': 'Tapez <span class="mono">ssh</span> et un espace : les serveurs de votre <span class="mono">~/.ssh/config</span>, chacun avec son adresse en regard. Pris sur macOS.',
    'ssh.demo.win': 'La même liste et le même <span class="mono">~/.ssh/config</span>, sous PowerShell — <span class="mono">atelier</span> compris, qui vient d’un <span class="mono">Include</span>. Pris sur Windows.',
    'ssh.demo.omarchy': 'Rien ne change sur Arch : le sélecteur lit <span class="mono">~/.ssh/config</span> et rien d’autre, il s’y comporte donc exactement comme dans les deux enregistrements ci-dessus. Aucun n’a encore été pris sur Omarchy.',
    'ssh.what.eyebrow': 'Ce qui est complété',
    'ssh.what.h2': 'Trois commandes, et trois choses à savoir.',
    'ssh.what.1h': 'Trois fournisseurs, pas un seul',
    'ssh.what.1p': '<span class="mono">ssh</span>, <span class="mono">scp</span> et <span class="mono">sftp</span> sont trois fournisseurs distincts, et non un seul répondant à trois noms. Ils lisent le même fichier, mais chacun répond pour sa commande — c’est ainsi que <span class="mono">scp</span> peut se comporter autrement que les deux autres.',
    'ssh.what.2h': 'Aucun verbe, donc le catalogue vient dès l’espace',
    'ssh.what.2p': '<span class="mono">brew install fire</span> a besoin de sa sous-commande avant que le catalogue veuille dire quelque chose. <span class="mono">ssh</span>, non : l’opérande suit directement le nom de la commande, donc les serveurs apparaissent dès l’espace, sans rien avoir tapé d’autre.',
    'ssh.what.3h': '<span class="mono">scp</span> insère un deux-points',
    'ssh.what.3p': 'Choisir <span class="mono">nas</span> derrière <span class="mono">scp</span> insère <span class="mono">nas:</span>, deux-points collé. Ce n’est pas cosmétique — sans lui, la commande s’exécute quand même, et fait tout autre chose.',
    'ssh.what.local': 'Les deux lignes ci-dessous diffèrent d’un caractère, et une seule atteint un serveur. La première copie <span class="mono">rapport.pdf</span> vers un fichier <strong>local</strong> nommé <span class="mono">nas</span>, dans le répertoire courant — valide, silencieuse, et fausse.',
    'ssh.what.remote': 'La seconde est celle que vous vouliez. <span class="mono">scp</span> reçoit donc le deux-points, et les deux autres non : <span class="mono">ssh nas</span> et <span class="mono">sftp nas</span> veulent le nom nu.',
    'ssh.file.eyebrow': 'Ce qu’il lit',
    'ssh.file.h2': 'Un fichier, et rien d’autre.',
    'ssh.file.lede': '<span class="mono">~/.ssh/config</span>. Pas de <span class="mono">known_hosts</span>, pas de <span class="mono">/etc/ssh/ssh_config</span>, pas de réseau — jigger n’ouvre jamais de connexion, et ne demande jamais rien à un serveur.',
    'dia4.alt': 'Trois colonnes : à gauche ~/.ssh/config et l’Include qui charge un second fichier ; au centre les six hôtes proposés par la fenêtre, atelier compris puisque l’Include est suivi ; à droite les deux motifs, Host *.exemple.net et Host *, barrés — un motif n’est pas un serveur, et n’est jamais proposé.',
    'dia4.col1': 'CE QU’IL LIT',
    'dia4.col2': 'CE QUE LA FENÊTRE PROPOSE',
    'dia4.col3': 'CE QU’IL ÉCARTE',
    'dia4.follow': 'les inclusions sont suivies',
    'dia4.inc': 'un second fichier, un hôte de plus',
    'dia4.offered': 'les six hôtes proposés',
    'dia4.atelier': 'atelier vient du second fichier',
    'dia4.proof': 'l’Include est ce qui l’amène ici',
    'dia4.dropped': 'les deux motifs, jamais proposés',
    'dia4.why': 'un nom avec * ? ou ! n’est pas un serveur',
    'dia4.why2': 'le proposer insérerait un nom injoignable',
    'dia4.never': 'ce qu’il ne lit jamais',
    'dia4.never2': 'et aucun réseau, jamais',
    'dia4.reread': 'relu à chaque frappe, rien en cache',
    'ssh.file.fixture': 'Le fichier derrière les deux enregistrements contient un <span class="mono">Include</span>, six hôtes et deux motifs. L’<span class="mono">Include</span> tire <span class="mono">atelier</span> d’un second fichier : c’est ainsi que les enregistrements montrent que les inclusions sont suivies. Les six — <span class="mono">passerelle</span>, <span class="mono">nas</span>, <span class="mono">proxmox</span>, <span class="mono">omarchy</span>, <span class="mono">windows</span> et <span class="mono">atelier</span> — sont ceux que la fenêtre propose, chacun avec son <span class="mono">HostName</span> à droite quand il diffère du nom. Les deux motifs, <span class="mono">Host *.exemple.net</span> et <span class="mono">Host *</span>, n’apparaissent jamais : un nom contenant <span class="mono">*</span>, <span class="mono">?</span> ou <span class="mono">!</span> n’est pas un serveur, et le proposer insérerait quelque chose à quoi on ne peut pas se connecter.',
    'ssh.file.reread': 'Tout est relu à chaque frappe. Il n’y a ni cache ni préchauffage : lire quelques fragments de configuration coûte une milliseconde. SSH n’est pas seul à n’avoir rien à tenir — scoop non plus, pour une autre raison : son catalogue est déjà étalé sur le disque, un manifeste par paquet, et le lire coûte moins cher que le mettre en cache.',
    'ssh.file.silence': 'Sur une machine sans <span class="mono">~/.ssh/config</span>, rien n’apparaît du tout — pas de fenêtre, pas de cadre vide, pas de « aucun candidat ». Idem quand rien ne correspond à ce que vous avez tapé. Un fournisseur au catalogue vide ne dessine aucun cadre.',
    'ssh.cta.eyebrow': 'Rien à ajouter',
    'ssh.cta.h2': 'Il est livré avec jigger.',
    'ssh.cta.lede': 'Il n’y a pas d’installation séparée ni de réglage à activer : le sélecteur vit dans le même binaire et le même greffon de shell que le reste. <span class="mono">JIGGER_COMMANDS</span> liste les commandes qui ouvrent la fenêtre — retirez-en <span class="mono">ssh</span>, <span class="mono">scp</span> et <span class="mono">sftp</span> et le sélecteur disparaît.',
    'ssh.cta.link': 'Télécharger, brancher, utiliser <span class="arw">→</span>',
    'hero.eyebrow': 'macOS · Windows · Linux',
    'hero.h1': 'Votre gestionnaire de paquets,<br><em>pendant que vous tapez.</em>',
    'hero.lede': 'Un cadre apparaît sous votre invite et suit chaque frappe. Et une seule syntaxe pour Homebrew, winget, scoop et pacman, au lieu d’une par gestionnaire.',
    'hero.download': 'Télécharger <span class="arw">→</span>',
    'hero.see': 'Le voir à l’œuvre <span class="arw">→</span>',
    'home.demo.mac01': 'brew install fire — le cadre arrive tout seul et se resserre à chaque lettre. Pris sur macOS.',
    'home.demo.win01': 'winget install fire — le même cadre et les mêmes touches, un autre catalogue. Pris sur Windows.',
    'home.demo.omarchy': 'Sur Arch — Omarchy compris — le même cadre répond pour <span class="mono">pacman</span> et <span class="mono">yay</span>, les dépôts et l’AUR dans une seule liste. Aucun enregistrement n’en existe encore : les deux ci-dessus ont été pris sur les machines qui étaient joignables.',
    'popup.eyebrow': 'La fenêtre',
    'popup.h2': 'Elle suit votre frappe.',
    'popup.lede': 'Tapez une commande de gestionnaire : le cadre apparaît seul et se resserre à chaque lettre. Rien à déclencher, rien à retenir.',
    'dia2.alt': 'La même ligne tapée dans trois états — fire, puis fireb, puis firebi — et le cadre qui se resserre à côté : sept candidats, puis trois, puis un seul. Rien ne l’ouvre, la liste suit simplement la frappe.',
    'dia2.eyebrow': 'PENDANT QUE VOUS TAPEZ',
    'dia2.note': 'rien à presser : le cadre s’ouvre seul et se resserre lettre après lettre',
    'dia2.c1': 'sept candidats',
    'dia2.c2': 'une lettre de plus, trois restent',
    'dia2.c3': 'une lettre de plus, un seul reste — ⇥ l’insère',
    'popup.k1': 'insère le candidat courant',
    'popup.k2': 'entre dans la liste',
    'popup.k3': 'remonte, puis rend le clavier au shell',
    'popup.k4': 'referme pour cette ligne',
    'popup.k5': 'filtre par expression rationnelle au lieu du préfixe',
    'popup.arrows': '<strong>Vos flèches ne sont jamais confisquées.</strong> Tant que la fenêtre n’a pas le clavier, <kbd>↑</kbd> et <kbd>↓</kbd> restent l’historique du shell — là où elles doivent être.',
    'popup.ssh': '<strong>Pas seulement les gestionnaires de paquets.</strong> Tapez <span class="mono">ssh</span>, <span class="mono">scp</span> ou <span class="mono">sftp</span> : la fenêtre propose les hôtes de votre <span class="mono">~/.ssh/config</span>, chacun avec son <span class="mono">HostName</span> en regard. jigger n’ouvre jamais la connexion — il complète la ligne que vous lancerez vous-même, et ne montre rien du tout quand il n’a aucun hôte à proposer. <a href="/ssh.html">Le sélecteur SSH a sa propre page <span class="arw">→</span></a>',
    'facade.eyebrow': 'Une syntaxe',
    'facade.h2': 'Un vocabulaire, tous les dialectes.',
    'facade.lede': '<span class="mono">jg install fd</span> atteint celui de Homebrew, winget, scoop ou pacman qui connaît <span class="mono">fd</span> — sans que vous ayez à savoir lequel.',
    'facade.verbs': 'Les sept premiers marchent partout. Les autres n’existent que là où le gestionnaire les connaît — demander <span class="mono">cleanup</span> à winget, ou <span class="mono">source</span> à pacman, échoue proprement, en nommant qui saurait le faire.',
    'dia3.alt': 'Un verbe au centre, jg install fd, et quatre gestionnaires autour. Une seule flèche pleine atteint celui dont le catalogue contient le nom ; les trois autres restent éteintes. En dessous, le cas où deux gestionnaires connaissent le même nom : le sélecteur s’ouvre et vous tranchez — rien n’est réglé à votre place.',
    'dia3.eyebrow': 'UN VERBE, PLUSIEURS PORTES',
    'dia3.note': 'routé par ce que contiennent les catalogues, jamais par un réglage',
    'dia3.knows': 'il connaît le nom — il exécute',
    'dia3.absent': 'pas ce nom dans son catalogue',
    'dia3.tag': 'UN VERBE',
    'dia3.center': 'qui connaît ce nom ?',
    'dia3.tie': 'DEUX FOIS LE MÊME NOM',
    'dia3.tienote': 'pas le même logiciel',
    'dia3.pickertag': 'LE SÉLECTEUR',
    'dia3.picker1': 'il s’ouvre, et vous tranchez',
    'dia3.picker2': 'rien n’est décidé à votre place',
    'dia3.picker3': '--pm tranche dans un script',
    'dia3.never': 'Aucun réglage ne change ça.',
    'dia3.never2': 'c’est vous qui choisissez',
    'facade.choice': '<strong>Jamais de choix automatique.</strong> Si un seul gestionnaire connaît le nom, il gagne sans rien demander. Si plusieurs le connaissent, le sélecteur s’ouvre et vous tranchez — deux paquets qui portent le même nom ne sont pas forcément le même logiciel. Aucun réglage ne change ça ; <span class="mono">--pm</span> est là pour trancher dans un script.',
    'facade.arch': 'Sous Arch, <span class="mono">pacman</span> et <span class="mono">yay</span> sont deux portes sur la même base : yay pilote, pacman lit. Quel que soit celui que vous avez, <span class="mono">jg</span> liste vos paquets une fois — jamais deux.',
    'dia.eyebrow': 'Le fonctionnement',
    'dia.h2': 'Trois canaux, un seul binaire.',
    'dia.lede': 'jigger s’intercale entre votre shell et vos gestionnaires de paquets par trois chemins distincts : il lit leurs catalogues pour compléter ce que vous tapez, traduit un vocabulaire unique vers celui de chacun, et compte les mises à jour en attente en tâche de fond pour votre invite.',
    'dia.alt': 'Trois bandes : votre shell en haut, jigger au milieu, les cinq gestionnaires de paquets et les commandes ssh en bas, reliés par trois canaux — la complétion qui lit les catalogues, la façade qui exécute une commande traduite, et l’invite qui compte les mises à jour en tâche de fond.',
    'dia.shell': 'VOTRE SHELL',
    'dia.a1': '↓ chaque frappe, relayée',
    'dia.a2': '↑ un cadre à afficher',
    'dia.b1': '↓ un verbe, des noms',
    'dia.b2': '↑ la sortie, telle quelle',
    'dia.c1': '↓ le hook lit une ligne',
    'dia.c2': '↑ version et compteurs',
    'dia.sub': 'un binaire Go autonome, démarrage quasi instantané',
    'dia.nodaemon': 'AUCUN DÉMON · AUCUN RÉSEAU',
    'dia.read': 'LECTURE',
    'dia.read1': 'Le premier mot décide qui parle.',
    'dia.read2': 'Sans état : l’index reste au shell.',
    'dia.run': 'EXÉCUTION',
    'dia.run1': 'Douze verbes vers tous les dialectes.',
    'dia.run2': 'Routé par catalogue, jamais par réglage.',
    'dia.state': 'ÉTAT',
    'dia.state1': 'Compte ce qui attend, en tâche de fond.',
    'dia.state2': 'Jamais sur le chemin critique de l’invite.',
    'dia.a3': '↓ lit catalogue et installés',
    'dia.a4': '↑ candidats, badges, corrections',
    'dia.b3': '↓ la commande traduite',
    'dia.b4': '↑ flux brut, code de retour',
    'dia.c3': '↓ interrogé en tâche de fond',
    'dia.c4': '↑ version et nombre en attente',
    'dia.mgrs': 'LES GESTIONNAIRES',
    'dia.mgrsnote': 'chacun répond à sa façon — jigger s’adapte, eux ne changent pas',
    'dia.m1': 'catalogue',
    'dia.m1b': 'en cache 24 h',
    'dia.m2': 'il lit,',
    'dia.m2b': 'yay pilote',
    'dia.m3': 'les dépôts',
    'dia.m3b': 'et l’AUR',
    'dia.m4': '14 401 ids,',
    'dia.m4b': 'en cache',
    'dia.m5': 'tout sur disque,',
    'dia.m5b': 'sans cache',
    'dia.m6': 'jamais exécuté',
    'dia.m6b': 'par jigger',
    'dia.caption': '<strong>Bleu</strong> — la complétion lit, elle n’exécute rien. <strong>Ambre</strong> — la façade traduit un verbe, puis laisse le gestionnaire faire le travail et relaie sa sortie sans y toucher. <strong>Vert</strong> — l’invite : en pointillés là où ça tourne en tâche de fond, en trait plein là où le hook se contente de lire une ligne, sans rien coûter.',
    'guar.eyebrow': 'Ce qu’il garantit',
    'guar.h2': 'Il ne se met pas en travers.',
    'guar.1h': 'Rien de lent pendant la frappe',
    'guar.1p': 'Les catalogues sont en cache et lus sur disque. Une frappe n’attend jamais un gestionnaire de paquets.',
    'guar.2h': 'La sortie de votre gestionnaire, intacte',
    'guar.2p': 'Invites, barres de progression, demandes d’élévation : jigger les relaie telles quelles au lieu de les digérer.',
    'guar.3h': 'Fait aussi pour les tubes',
    'guar.3p': 'Chaque verbe qui liste accepte <span class="mono">--json</span> : un script n’a jamais à lire un tableau écrit pour des humains.',
    'prompt.eyebrow': 'Dans votre invite',
    'prompt.h2': 'Un bloc dans votre invite, si vous en voulez un.',
    'prompt.lede': 'Des segments prêts à coller pour oh-my-posh et starship : la version de votre gestionnaire, et le nombre de paquets en attente — comptés à part, et jamais sur le chemin critique de l’invite.',
    'prompt.how': 'Posez <span class="mono">JIGGER_PROMPT=1</span> avant de charger le greffon, puis collez le segment de votre invite et de votre gestionnaire — les versions brew, pacman et Windows sont toutes livrées avec jigger.',
    'coc.eyebrow': 'À côté',
    'coc.h2': 'Et quand vous préférez voir que taper.',
    'coc.lede': '<strong>Cocktails</strong> est l’application macOS de la même famille : vos paquets, leurs dépendances et vos mises à jour, d’un coup d’œil. jigger fonctionne très bien sans elle — les deux sont indépendants.',
    'coc.link': 'Voir le site de Cocktails <span class="arw">→</span>',
    'cta.eyebrow': 'Installer',
    'cta.h2': 'Deux lignes, puis une dans votre shell.',
    'cta.lede': 'Homebrew sur macOS et Linux, scoop sur Windows, <span class="mono">go install</span> partout ailleurs. Puis une ligne <span class="mono">source</span> dans <span class="mono">~/.zshrc</span>, ou un <span class="mono">Import-Module</span> dans votre profil PowerShell.',
    'cta.where': 'Les commandes vivent sur une page à elles, à côté de quoi vérifier quand rien n’apparaît, des touches, et des réglages qui servent dès le premier jour.',
    'cta.link': 'Télécharger, brancher, utiliser <span class="arw">→</span>',
    'cta.source': 'Lire le code',
    'foot.made': 'Logiciel libre, Apache 2.0. Développé à ciel ouvert.',
    'foot.repo': 'Le dépôt',
    'foot.guide': 'Premiers pas',
    'foot.changelog': 'Journal des versions'
  };
  /* --- /FR --- */

  /* Le <title> de chaque page porte son propre `data-i18n` et suit donc le même
     chemin que le reste : l'anglais reste écrit dans la balise, le français vient
     du dictionnaire. Une table de titres dans ce fichier n'en connaissait qu'un, et
     l'appliquait aux trois pages — onglets, favoris et indexation confondus. */
  var i18nEls = Array.prototype.slice.call(document.querySelectorAll('[data-i18n]'));
  i18nEls.forEach(function (el) { el.setAttribute('data-en', el.innerHTML); });

  var lang = 'en';

  function setLang(l) {
    lang = (l === 'fr') ? 'fr' : 'en';
    docEl.lang = lang;
    i18nEls.forEach(function (el) {
      var k = el.getAttribute('data-i18n');
      /* Repli sur l'anglais : une clé française absente laisse le texte d'origine. */
      el.innerHTML = (lang === 'fr' && FR[k] != null) ? FR[k] : el.getAttribute('data-en');
    });
    document.querySelectorAll('.lang-toggle button').forEach(function (b) {
      var on = b.getAttribute('data-lang') === lang;
      b.setAttribute('aria-pressed', on ? 'true' : 'false');
      b.classList.toggle('on', on);
    });
    if (boutons) {
      boutons.forEach(function (b) { etiqueter(b, b.getAttribute('data-state')); });
    }
    try { localStorage.setItem('jigger-lang', lang); } catch (e) {}
  }

  document.querySelectorAll('.lang-toggle button').forEach(function (b) {
    b.addEventListener('click', function () { setLang(b.getAttribute('data-lang')); });
  });

  /* Langue initiale : le choix mémorisé, sinon le navigateur, sinon l'anglais —
     comme le binaire depuis la v0.9.0. */
  var saved = null;
  try { saved = localStorage.getItem('jigger-lang'); } catch (e) {}
  var initial = saved || ((navigator.language || 'en').toLowerCase().indexOf('fr') === 0 ? 'fr' : 'en');

  /* --- démonstrations ---------------------------------------------------
     Les vidéos ne portent pas leur src : app.js le pose au moment où la
     démonstration doit jouer. Deux raisons — ne pas télécharger les
     enregistrements du système que le lecteur n'a pas choisi, et ne rien
     démarrer chez qui a demandé moins d'animations à son système. */
  var immobile = false;
  try {
    immobile = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  } catch (e) {}

  /* Préférence explicite du lecteur, qui l'emporte sur le réglage système.
     Sans elle, prefers-reduced-motion décidait SEUL — et un lecteur qui a
     demandé « moins d'animations » n'avait alors aucun moyen de voir les
     enregistrements : la page se réduisait à des images fixes, sans le dire.
     C'est tout le contenu illustré du site qui lui restait invisible. */
  var prefAnim = null;
  try { prefAnim = localStorage.getItem('jigger-anim'); } catch (e) {}

  function animationsAuto() {
    if (prefAnim === 'on') { return true; }
    if (prefAnim === 'off') { return false; }
    return !immobile;
  }

  /* Le libellé du bouton ne passe pas par data-i18n : il change avec l'état
     autant qu'avec la langue, ce que le remplacement d'innerHTML ne sait pas
     faire. setLang le rappelle, plus bas. */
  var ETIQ = {
    en: { paused: 'Play this recording', playing: 'Pause this recording' },
    fr: { paused: 'Lire cet enregistrement', playing: 'Mettre en pause' }
  };

  function etiqueter(b, etat) {
    b.setAttribute('data-state', etat);
    b.setAttribute('aria-label', (ETIQ[lang] || ETIQ.en)[etat]);
    /* U+FE0E force la présentation TEXTE : sans lui, macOS rend ces deux
       caractères en émoji couleur, hors de la palette du cadre. */
    b.firstChild.textContent = (etat === 'playing') ? '\u23F8\uFE0E' : '\u25B6\uFE0E';
  }

  function lire(v) {
    if (!v.src) { v.src = v.getAttribute('data-src'); }
    var p = v.play();
    if (p && p.catch) { p.catch(function () {}); }
  }

  /* Un bouton par démonstration, pour tout le monde : une vidéo qui tourne en
     boucle sans moyen de l'arrêter est un défaut d'accessibilité (WCAG 2.2.2).
     Il s'efface quand la lecture est en cours, et revient au survol ou au
     clavier. */
  var boutons = [];
  Array.prototype.forEach.call(document.querySelectorAll('.demo-frame'), function (cadre) {
    var v = cadre.querySelector('video[data-src]');
    if (!v) { return; }
    var b = document.createElement('button');
    b.type = 'button';
    b.className = 'demo-play';
    b.appendChild(document.createElement('span'));
    cadre.appendChild(b);
    boutons.push(b);

    b.addEventListener('click', function () {
      if (!v.paused) { v.pause(); return; }
      if (animationsAuto()) { lire(v); return; }
      /* Premier « lire » d'un lecteur en mouvement réduit : on retient le
         choix et on lance tout ce qui est visible, sinon il devrait le refaire
         sur chaque démonstration et sur chaque page. Le réglage système n'est
         pas contourné — il est arbitré par un geste explicite. */
      prefAnim = 'on';
      try { localStorage.setItem('jigger-anim', 'on'); } catch (e) {}
      activerDemos();
    });

    v.addEventListener('play', function () { etiqueter(b, 'playing'); });
    v.addEventListener('pause', function () { etiqueter(b, 'paused'); });
    etiqueter(b, 'paused');
  });

  function activerDemos() {
    Array.prototype.forEach.call(document.querySelectorAll('video[data-src]'), function (v) {
      /* offsetParent vaut null quand un ancêtre est en display:none — c'est
         ainsi qu'on sait qu'une démonstration appartient au système inactif.
         On la met en pause au lieu de simplement l'ignorer : une vidéo déjà
         lancée continue de tourner une fois masquée, et après un aller-retour
         d'un système à l'autre toutes les démonstrations de la page
         décoderaient en même temps, dont celles que personne ne regarde. */
      if (v.offsetParent === null) { v.pause(); return; }
      if (animationsAuto()) { lire(v); }
    });
  }

  /* --- sélecteur de système ---------------------------------------------
     Une préférence de site, pas de page : elle vaut pour les trois. L'ordre
     de décision — l'URL d'abord (un lien partagé doit imposer ce qu'il
     montre), puis le choix mémorisé, puis la plateforme du navigateur. */
  var SYSTEMES = ['macos', 'windows', 'omarchy'];

  function setOs(os) {
    /* Liste blanche plutôt que ternaire : à trois systèmes, « tout ce qui
       n'est pas Windows est macOS » enverrait un ?os=linux partagé sur la
       mauvaise page sans que rien ne le signale. Une valeur inconnue retombe
       sur macOS, qui reste le cas le plus fréquent. */
    if (SYSTEMES.indexOf(os) === -1) { os = 'macos'; }
    docEl.setAttribute('data-os', os);
    Array.prototype.forEach.call(document.querySelectorAll('.os-toggle button'), function (b) {
      var on = b.getAttribute('data-os') === os;
      b.setAttribute('aria-pressed', on ? 'true' : 'false');
      b.classList.toggle('on', on);
    });
    try { localStorage.setItem('jigger-os', os); } catch (e) {}
    activerDemos();
  }

  Array.prototype.forEach.call(document.querySelectorAll('.os-toggle button'), function (b) {
    b.addEventListener('click', function () { setOs(b.getAttribute('data-os')); });
  });

  var osUrl = null;
  try { osUrl = new URLSearchParams(location.search).get('os'); } catch (e) {}
  var osMemo = null;
  try { osMemo = localStorage.getItem('jigger-os'); } catch (e) {}
  /* Android annonce « Linux » lui aussi, et n'a ni pacman ni yay : on l'écarte
     avant de conclure, sans quoi un visiteur sur téléphone verrait les
     commandes d'Arch. Ce qui reste sans correspondance retombe sur macOS. */
  var signature = navigator.platform || navigator.userAgent || '';
  var osNav = 'macos';
  if (/win/i.test(signature)) {
    osNav = 'windows';
  } else if (/linux/i.test(signature) && !/android/i.test(signature)) {
    osNav = 'omarchy';
  }
  setOs(osUrl || osMemo || osNav);

  setLang(initial);
  activerDemos();
})();
