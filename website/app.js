/* jigger — bascule de langue (fichier externe : compatible CSP 'self'). */
(function () {
  var docEl = document.documentElement;

  /* --- FR --- */
  var FR = {
    'nav.popup': 'La fenêtre',
    'nav.facade': 'Une syntaxe',
    'nav.install': 'Installer',
    'hero.eyebrow': 'macOS · Windows · Linux',
    'hero.h1': 'Votre gestionnaire de paquets,<br><em>pendant que vous tapez.</em>',
    'hero.lede': 'Un cadre apparaît sous votre invite et suit chaque frappe. Et pour Homebrew, winget et scoop, une seule syntaxe au lieu de trois.',
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
    'facade.eyebrow': 'Une syntaxe',
    'facade.h2': 'Trois gestionnaires, un vocabulaire.',
    'facade.lede': '<span class="mono">jg install fd</span> atteint celui de Homebrew, winget ou scoop qui connaît <span class="mono">fd</span> — sans que vous ayez à savoir lequel.',
    'facade.verbs': 'Les sept premiers marchent partout. Les autres n’existent que là où le gestionnaire les connaît — demander <span class="mono">cleanup</span> à winget échoue proprement, en nommant qui saurait le faire.',
    'facade.choice': '<strong>Jamais de choix automatique.</strong> Si un seul gestionnaire connaît le nom, il gagne sans rien demander. Si plusieurs le connaissent, le sélecteur s’ouvre et vous tranchez — deux paquets qui portent le même nom ne sont pas forcément le même logiciel. Aucun réglage ne change ça ; <span class="mono">--pm</span> est là pour trancher dans un script.',
    'guar.eyebrow': 'Ce qu’il garantit',
    'guar.h2': 'Il ne se met pas en travers.',
    'guar.1h': 'Rien de lent pendant la frappe',
    'guar.1p': 'Les catalogues sont en cache et lus sur disque. Une frappe n’attend jamais un gestionnaire de paquets.',
    'guar.2h': 'La sortie de votre gestionnaire, intacte',
    'guar.2p': 'Invites, barres de progression, demandes d’élévation : jigger les relaie telles quelles au lieu de les digérer.',
    'guar.3h': 'Fait aussi pour les tubes',
    'guar.3p': 'Chaque verbe qui liste accepte <span class="mono">--json</span> : un script n’a jamais à lire un tableau écrit pour des humains.',
    'prompt.eyebrow': 'Dans votre invite',
    'prompt.h2': 'Un bloc Homebrew, si vous en voulez un.',
    'prompt.lede': 'Des segments prêts à coller pour oh-my-posh et starship : la version de brew, et le nombre de formules et de casks en attente.',
    'prompt.how': 'Posez <span class="mono">JIGGER_PROMPT=1</span> avant de charger le greffon, puis collez le segment de votre invite — les deux sont livrés avec jigger.',
    'coc.eyebrow': 'À côté',
    'coc.h2': 'Et quand vous préférez voir que taper.',
    'coc.lede': '<strong>Cocktails</strong> est l’application macOS de la même famille : vos paquets, leurs dépendances et vos mises à jour, d’un coup d’œil. jigger fonctionne très bien sans elle — les deux sont indépendants.',
    'coc.link': 'Voir le site de Cocktails <span class="arw">→</span>',
    'install.eyebrow': 'Installer',
    'install.h2': 'Trois lignes, sur chacun des trois.',
    'install.mac': 'macOS et Linux — Homebrew',
    'install.win': 'Windows — depuis le clone',
    'install.winp': 'Le script compile, puis met <span class="mono">jigger</span> à portée : un shim scoop si scoop est là, une copie dans votre répertoire utilisateur sinon. Il n’existe pas encore de paquet winget ou scoop pour jigger.',
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
    en: 'jigger — one syntax for Homebrew, winget and scoop',
    fr: 'jigger — une syntaxe pour Homebrew, winget et scoop'
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
