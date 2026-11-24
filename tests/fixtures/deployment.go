package fixture
import("crypto/sha256";"encoding/hex";"fmt";"zyposoft.local/operations/packages/local-policy";"zyposoft.local/operations/packages/signing-go")
// Deployment manufactures only deterministic test fixtures and a separately provisioned admission policy.
func(s *Site)Deployment(seq int,key,lock,release,category,grantLabel string)policy.Bundle{
 b:=s.Job(seq,true,[]string{grantLabel},"mock-job")
 manifest:=Doc(b.Manifest);m:=manifest["body"].(signing.Object)
 m["product_id"]=key;m["release_id"]=release;m["required_locks"]=[]string{lock}
 m["health_probes"]=[]signing.Object{{"name":"app-readiness","probe_type":"HTTP","target":"/local/v1/readiness"}}
 b.Manifest=s.Sign(manifest,"mock-release");artifact,err:=policy.Digest(b.Manifest);if err!=nil{panic(err)}
 job:=Doc(b.Job);j:=job["body"].(signing.Object);p:=j["params"].(signing.Object)
 p["product_key"]=key;p["manifest_sha256"]=artifact;b.Job=s.Sign(job,"mock-job")
 backup:=Doc(b.Backup);e:=backup["body"].(signing.Object)
 hash:=sha256.Sum256([]byte(fmt.Sprintf("synthetic backup %s %d",key,seq)));backupID:=hex.EncodeToString(hash[:]);e["backup_manifest_sha256"]=backupID;e["dataset_categories"]=[]string{category};e["backup_age_seconds"]=0
 b.Backup=s.Sign(backup,"mock-backup")
 grant:=Doc(b.Grants[0]);g:=grant["body"].(signing.Object);g["resource_scope"]=[]string{key};g["target_digest_sha256"]=artifact
 nested,err:=signing.SignAttestation("MC-09",JSON(grant),s.Config.Trust,"mock-local",s.priv["mock-local"]);if err!=nil{panic(err)};g["site_authoriser_attestation"]=nested;b.Grants[0]=s.Sign(grant,"mock-grant")
 s.Config.TestArtifacts[artifact]=key;if s.Config.BackupForJob==nil{s.Config.BackupForJob=map[string]policy.BackupRequirement{}}
 s.Config.BackupForJob[j["job_id"].(string)]=policy.BackupRequirement{Product:key,Artifact:artifact,BackupManifest:backupID,Categories:[]string{category}}
 return b
}
