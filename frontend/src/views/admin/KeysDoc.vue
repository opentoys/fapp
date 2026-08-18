<script setup lang="ts">
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'
import { Alert } from '../../components/ui/alert'
import { Badge } from '../../components/ui/badge'
import { useI18n } from '../../composables/useI18n'

const { t } = useI18n()

const base = () => window.location.origin

const endpoints = [
  {
    method: 'POST',
    path: '/api/v1/keys/{id}/files',
    scope: 'run',
    desc: 'apiDoc.epPresignDesc',
    docs: [
      { name: 'file_name', type: 'string', required: true, desc: 'apiDoc.fieldApk' },
      { name: 'sha256', type: 'string', required: true, desc: 'apiDoc.fieldSha256' },
      { name: 'file_size', type: 'int', required: true, desc: 'apiDoc.fieldFileSize' },
    ],
    example: `SHA=$(shasum -a 256 app.apk | cut -d' ' -f1)\nSIZE=$(stat -f%z app.apk)\ncurl -X POST "${base()}/api/v1/keys/123/files?apikey=dk_xxx" \\\n  -H "Content-Type: application/json" \\\n  -d "{\\"file_name\\":\\"app.apk\\",\\"sha256\\":\\"$SHA\\",\\"file_size\\":$SIZE}"`,
    resp: `{
  "code": 0,
  "data": {
    "key": "disapp/123/0/3f2a…c9b1_1048576.apk",
    "url": "…signed upload url…"
  },
  "msg": "ok"
}`,
  },
  {
    method: 'POST',
    path: '/api/v1/keys/{id}/versions',
    scope: 'run',
    desc: 'apiDoc.epUploadDesc',
    docs: [
      { name: 'key', type: 'string', required: true, desc: 'apiDoc.fieldKey' },
      { name: 'file_name', type: 'string', required: true, desc: 'apiDoc.fieldApk' },
      { name: 'file_size', type: 'int', required: true, desc: 'apiDoc.fieldFileSize' },
      { name: 'sha256', type: 'string', required: true, desc: 'apiDoc.fieldSha256' },
      { name: 'version_name', type: 'string', required: true, desc: 'apiDoc.fieldVersionName' },
      { name: 'version_code', type: 'int', required: true, desc: 'apiDoc.fieldVersionCode' },
    ],
    example: `SHA=$(shasum -a 256 app.apk | cut -d' ' -f1)\nSIZE=$(stat -f%z app.apk)\nKEY=$(curl -s -X POST "${base()}/api/v1/keys/123/files?apikey=dk_xxx" \\\n  -H "Content-Type: application/json" \\\n  -d "{\\"file_name\\":\\"app.apk\\",\\"sha256\\":\\"$SHA\\",\\"file_size\\":$SIZE}" | jq -r .data.key)\n# push bytes to the presigned url, then create the version\ncurl -X POST "${base()}/api/v1/keys/123/versions?apikey=dk_xxx" \\\n  -H "Content-Type: application/json" \\\n  -d "{\\"key\\":\\"$KEY\\",\\"file_name\\":\\"app.apk\\",\\"file_size\\":$SIZE,\\"sha256\\":\\"$SHA\\",\\"version_name\\":\\"1.0.0\\",\\"version_code\\":1}"`,
    resp: `{
  "code": 0,
  "data": {
    "id": 9,
    "release_type": "",
    "platform": "android",
    "version_name": "1.0.0",
    "version_code": 1,
    "file_type": "apk",
    "file_name": "app.apk",
    "file_size": 1048576,
    "appid": "com.example.app",
    "sha256": "9f2c…c0",
    "download_count": 0,
    "install_count": 0
  },
  "msg": "ok"
}`,
  },
  {
    method: 'POST',
    path: '/api/v1/keys/{id}/current',
    scope: 'run',
    desc: 'apiDoc.epSetCurrentDesc',
    docs: [
      { name: 'version_id', type: 'int', required: true, desc: 'apiDoc.fieldVersionId' },
    ],
    example: `curl -X POST "${base()}/api/v1/keys/123/current?apikey=dk_xxx" \\\n  -H "Content-Type: application/json" \\\n  -d '{"version_id": 5}'`,
    resp: `{
  "code": 0,
  "data": {
    "id": 123,
    "name": "Example",
    "platform": "android",
    "published": true,
    "current_version_id": 5
  },
  "msg": "ok"
}`,
  },
  {
    method: 'GET',
    path: '/api/v1/keys/{id}/versions',
    scope: 'run / read',
    desc: 'apiDoc.epListDesc',
    docs: [],
    example: `curl "${base()}/api/v1/keys/123/versions?apikey=dk_xxx"`,
    resp: `{
  "code": 0,
  "data": {
    "app": {
      "id": 123,
      "name": "Example",
      "platform": "android",
      "appid": "com.example.app"
    },
    "versions": [
      { "id": 9, "version_name": "1.0.0", "version_code": 1, "file_size": 1048576 }
    ]
  },
  "msg": "ok"
}`,
  },
  {
    method: 'GET',
    path: '/api/v1/keys/{id}/current',
    scope: 'run / read',
    desc: 'apiDoc.epCurrentDesc',
    docs: [],
    example: `curl "${base()}/api/v1/keys/123/current?apikey=dk_xxx"`,
    resp: `{
  "code": 0,
  "data": {
    "app": { "id": 123, "name": "Example", "platform": "android", "appid": "com.example.app", "current_version_id": 5 },
    "versions": [
      { "id": 5, "version_name": "2.0.0", "version_code": 2, "file_size": 2097152 }
    ]
  },
  "msg": "ok"
}`,
  },
  {
    method: 'GET',
    path: '/api/v1/keys/{id}/current/download',
    scope: 'run / read',
    desc: 'apiDoc.epDownloadDesc',
    docs: [],
    example: `curl "${base()}/api/v1/keys/123/current/download?apikey=dk_xxx"`,
    resp: `{
  "code": 0,
  "data": {
    "url": "http://your-host.com/api/v1/files/123/5/app.apk"
  },
  "msg": "ok"
}`,
  },
]

const scopeBadge = (s: string) => (s.includes('run') ? 'success' : 'secondary')
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 py-8 sm:px-6">
    <div class="mb-6">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('apiDoc.title') }}</h1>
      <p class="text-muted-foreground" v-html="t('apiDoc.intro')"></p>
    </div>

    <Alert class="mb-6">
      <p v-html="t('apiDoc.hostNote')"></p>
    </Alert>

    <div class="grid gap-6">
      <Card v-for="(ep, i) in endpoints" :key="i">
        <CardHeader>
          <div class="flex items-center gap-3">
            <Badge :variant="scopeBadge(ep.scope)">{{ ep.method }}</Badge>
            <CardTitle class="font-mono text-base">{{ ep.path }}</CardTitle>
            <Badge variant="outline" class="ml-auto">{{ ep.scope }}</Badge>
          </div>
        </CardHeader>
        <CardContent class="grid gap-4">
          <p class="text-sm text-muted-foreground" v-html="t(ep.desc)"></p>
          <Table v-if="ep.docs.length">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('apiDoc.colName') }}</TableHead>
                <TableHead>{{ t('apiDoc.colType') }}</TableHead>
                <TableHead>{{ t('apiDoc.colRequired') }}</TableHead>
                <TableHead>{{ t('apiDoc.colDesc') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="d in ep.docs" :key="d.name">
                <TableCell class="font-mono">{{ d.name }}</TableCell>
                <TableCell><code>{{ d.type }}</code></TableCell>
                <TableCell>{{ d.required ? t('common.yes') : t('common.no') }}</TableCell>
                <TableCell class="text-muted-foreground">{{ t(d.desc) }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <pre class="overflow-x-auto rounded-md bg-muted p-4 text-xs"><code>{{ ep.example }}</code></pre>
          <template v-if="ep.resp">
            <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{{ t('apiDoc.resp') }}</h4>
            <pre class="overflow-x-auto rounded-md bg-muted p-4 text-xs"><code>{{ ep.resp }}</code></pre>
          </template>
        </CardContent>
      </Card>
    </div>
  </div>
</template>