# Release Scripts

Este directorio contiene scripts para automatizar tareas relacionadas con releases.

## update-krew-manifest.py

Script Python que actualiza automáticamente el `krew-manifest.yaml` con:
- Nueva versión
- URLs actualizadas para los assets del release
- SHA256 checksums calculados automáticamente

### Uso manual

```bash
# Después de crear un release v0.1.3
python3 scripts/update-krew-manifest.py v0.1.3
```

### Uso automático

El script se ejecuta automáticamente en GitHub Actions después de cada release exitoso. Ver `.github/workflows/releases.yml` para más detalles.

### Requisitos

- Python 3.9+
- PyYAML: `pip install PyYAML`


## Flujo de trabajo

1. **Developer**: Hace push de un tag `v*.*.*`
2. **GitHub Actions**: 
   - Construye binarios para todas las plataformas
   - Crea release en GitHub
   - Sube assets (tar.gz, zip) con SHA256
   - **Ejecuta `update-krew-manifest.py`** automáticamente
   - Hace commit y push del manifest actualizado
3. **Result**: `krew-manifest.yaml` queda actualizado automáticamente

Ya no necesitas actualizar manualmente URLs y SHA256 checksums! 🎉