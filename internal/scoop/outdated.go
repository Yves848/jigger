package scoop

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Outdated compte les applications scoop dont le bucket annonce une autre version que
// celle installée. C'est ce que fait `scoop status`, mais sans démarrer PowerShell ni
// toucher au réseau : la réponse est celle des buckets tels qu'ils sont sur le disque —
// donc datée du dernier `scoop update`, exactement comme `brew outdated` l'est du
// dernier `brew update`.
func Outdated() (int, error) { return OutdatedIn(Root(), GlobalRoot()) }

// OutdatedIn est Outdated sur des racines données (tests). `root` fournit les buckets ;
// les applications sont comptées dans les deux racines.
func OutdatedIn(root, global string) (int, error) {
	apps, err := OutdatedAppsIn(root, global)
	if err != nil {
		return 0, err
	}
	return len(apps), nil
}

// AppObsolete est une application scoop dont le bucket annonce une autre version que
// celle installée.
type AppObsolete struct {
	Nom        string
	Installee  string
	Disponible string
	Bucket     string
}

// OutdatedApps rend les applications scoop obsolètes, dans les mêmes conditions que
// Outdated : lecture du disque, sans démarrer PowerShell ni toucher au réseau.
func OutdatedApps() ([]AppObsolete, error) { return OutdatedAppsIn(Root(), GlobalRoot()) }

// OutdatedAppsIn est OutdatedApps sur des racines données (tests). `root` fournit les
// buckets ; les applications sont recensées dans les deux racines.
func OutdatedAppsIn(root, global string) ([]AppObsolete, error) {
	if root == "" {
		return nil, nil
	}
	manifests := indexBuckets(root)

	var obsoletes []AppObsolete
	for _, racine := range []string{root, global} {
		if racine == "" {
			continue
		}
		for nom, version := range installed(racine) {
			if version == "" {
				continue
			}
			appDir := filepath.Join(racine, "apps", nom)
			bucket, tenue := infosInstallation(appDir)
			if tenue {
				continue // `scoop hold` : l'application ne bougera pas, ne la comptons pas
			}
			path := manifestPath(manifests, root, bucket, nom)
			if path == "" {
				continue // installée depuis une URL, ou bucket retiré depuis : rien à comparer
			}
			if v := versionManifeste(path); v != "" && v != version {
				obsoletes = append(obsoletes, AppObsolete{
					Nom:        nom,
					Installee:  version,
					Disponible: v,
					Bucket:     bucket,
				})
			}
		}
	}
	return obsoletes, nil
}

// indexBuckets rend, pour chaque application connue, le manifeste du premier bucket qui
// la contient (dans l'ordre de prioritaire). Il sert de recours quand l'installation ne
// dit pas de quel bucket elle vient.
func indexBuckets(root string) map[string]string {
	index := map[string]string{}
	buckets := map[string][]string{}
	dirs := map[string]string{}
	for _, dir := range bucketDirs(root) {
		b := bucketName(dir)
		dirs[b] = dir
		for _, nom := range manifests(dir) {
			buckets[nom] = append(buckets[nom], b)
		}
	}
	for nom, bs := range buckets {
		index[nom] = filepath.Join(dirs[prioritaire(bs)], nom+".json")
	}
	return index
}

// manifestPath rend le manifeste de référence d'une application : celui de son bucket
// d'origine si l'installation le nomme, sinon celui qu'on lui trouve.
func manifestPath(index map[string]string, root, bucket, nom string) string {
	if bucket != "" {
		for _, dir := range []string{
			filepath.Join(root, "buckets", bucket, "bucket", nom+".json"),
			filepath.Join(root, "buckets", bucket, nom+".json"),
		} {
			if _, err := os.Stat(dir); err == nil {
				return dir
			}
		}
	}
	return index[nom]
}

// infosInstallation lit apps/<nom>/current/install.json : le bucket d'origine, et si
// l'application est tenue (`scoop hold`).
func infosInstallation(appDir string) (bucket string, tenue bool) {
	data, err := os.ReadFile(filepath.Join(appDir, "current", "install.json"))
	if err != nil {
		return "", false
	}
	var doc struct {
		Bucket string `json:"bucket"`
		Hold   bool   `json:"hold"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", false
	}
	return doc.Bucket, doc.Hold
}
