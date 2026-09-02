/* jigger — bascule de langue (fichier externe : compatible CSP 'self'). */
(function () {
  var docEl = document.documentElement;

  /* --- FR --- */
  var FR = {
    'nav.popup': 'La fenêtre',
    'nav.facade': 'Une syntaxe',
    'nav.how': 'Le fonctionnement',
    'nav.install': 'Installer',
    'hero.eyebrow': 'macOS · Windows · Linux',
    'hero.h1': 'Votre gestionnaire de paquets,<br><em>pendant que vous tapez.</em>',
    'hero.lede': 'Un cadre apparaît sous votre invite et suit chaque frappe. Et pour Homebrew, winget, scoop et pacman, une seule syntaxe au lieu de quatre.',
    'hero.install': 'L’installer <span class="arw">→</span>',
    'hero.source': 'Lire le code',
    'popup.eyebrow': 'La fenêtre',
    'popup.h2': 'Elle suit votre frappe.',
    'popup.lede': 'Tapez une commande de gestionnaire : le cadre apparaît seul et se resserre à chaque lettre. Rien à déclencher, rien à retenir.',
    'popup.k1': 'insère le candidat courant',
    'popup.k2': 'entre dans la liste',
    'popup.k3': 'remonte, puis rend le clavier au shell',
    'popup.k4': 'referme pour cette ligne',
    'popup.arrows': '<strong>Vos flèches ne sont jamais confisquées.</strong> Tant que la fenêtre n’a pas le clavier, <kbd>↑</kbd> et <kbd>↓</kbd> restent l’historique du shell — là où elles doivent être.',
    'popup.ssh': '<strong>Pas seulement les gestionnaires de paquets.</strong> Tapez <span class="mono">ssh</span>, <span class="mono">scp</span> ou <span class="mono">sftp</span> : la fenêtre propose les hôtes de votre <span class="mono">~/.ssh/config</span>, chacun avec son <span class="mono">HostName</span> en regard. jigger n’ouvre jamais la connexion — il complète la ligne que vous lancerez vous-même, et ne montre rien du tout quand il n’a aucun hôte à proposer.',
    'facade.eyebrow': 'Une syntaxe',
    'facade.h2': 'Quatre dialectes, un vocabulaire.',
    'facade.lede': '<span class="mono">jg install fd</span> atteint celui de Homebrew, winget, scoop ou pacman qui connaît <span class="mono">fd</span> — sans que vous ayez à savoir lequel.',
    'facade.verbs': 'Les sept premiers marchent partout. Les autres n’existent que là où le gestionnaire les connaît — demander <span class="mono">cleanup</span> à winget, ou <span class="mono">source</span> à pacman, échoue proprement, en nommant qui saurait le faire.',
    'facade.choice': '<strong>Jamais de choix automatique.</strong> Si un seul gestionnaire connaît le nom, il gagne sans rien demander. Si plusieurs le connaissent, le sélecteur s’ouvre et vous tranchez — deux paquets qui portent le même nom ne sont pas forcément le même logiciel. Aucun réglage ne change ça ; <span class="mono">--pm</span> est là pour trancher dans un script.',
    'facade.arch': 'Sous Arch, <span class="mono">pacman</span> et <span class="mono">yay</span> sont deux portes sur la même base : yay pilote, pacman lit. Quel que soit celui que vous avez, <span class="mono">jg</span> liste vos paquets une fois — jamais deux.',
    'dia.eyebrow': 'Le fonctionnement',
    'dia.h2': 'Trois canaux, un seul binaire.',
    'dia.lede': 'jigger s’intercale entre votre shell et vos gestionnaires de paquets par trois chemins distincts : il lit leurs catalogues pour compléter ce que vous tapez, traduit un vocabulaire unique vers celui de chacun, et compte les mises à jour en attente en tâche de fond pour votre invite.',
    'dia.alt': 'Trois bandes : votre shell en haut, jigger au milieu, les six gestionnaires de paquets en bas, reliés par trois canaux — la complétion qui lit les catalogues, la façade qui exécute une commande traduite, et l’invite qui compte les mises à jour en tâche de fond.',
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
    'dia.run1': 'Douze verbes vers quatre dialectes.',
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
    'dia.m1': 'catalogue en cache 24 h',
    'dia.m2': 'il lit, yay pilote',
    'dia.m3': 'dépôts et AUR',
    'dia.m4': '14 401 ids, en cache',
    'dia.m5': 'tout sur disque, sans cache',
    'dia.m6': 'jamais exécuté par jigger',
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
    'install.eyebrow': 'Installer',
    'install.h2': 'Trois lignes, sur chacun des trois.',
    'install.mac': 'macOS et Linux — Homebrew',
    'install.win': 'Windows — scoop',
    'install.winp': 'Binaire précompilé, rien à compiler. Pour construire depuis un clone — en développant, ou pour une version pas encore publiée — le dépôt livre <span class="mono">install-windows.ps1</span>. Il n’existe pas de paquet winget.',
    'install.go': 'Toute plateforme — Go',
    'install.plugin': 'Reste à brancher le greffon dans votre shell : une ligne <span class="mono">source</span> dans <span class="mono">~/.zshrc</span>, un <span class="mono">Import-Module</span> dans votre profil PowerShell. Le guide déroule les deux, et ce qu’il faut faire quand rien n’apparaît.',
    'install.guide': 'Lire le guide de premiers pas <span class="arw">→</span>',
    'foot.made': 'Logiciel libre, Apache 2.0. Développé à ciel ouvert.',
    'foot.repo': 'Le dépôt',
    'foot.guide': 'Premiers pas',
    'foot.changelog': 'Journal des versions'
  };
  /* --- /FR --- */

  var TITLE = {
    en: 'jigger — one syntax for Homebrew, winget, scoop and pacman',
    fr: 'jigger — une syntaxe pour Homebrew, winget, scoop et pacman'
  };

  var i18nEls = Array.prototype.slice.call(document.querySelectorAll('[data-i18n]'));
  i18nEls.forEach(function (el) { el.setAttribute('data-en', el.innerHTML); });

  var lang = 'en';

  function setLang(l) {
    lang = (l === 'fr') ? 'fr' : 'en';
    docEl.lang = lang;
    document.title = TITLE[lang];
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
  setLang(initial);
})();
