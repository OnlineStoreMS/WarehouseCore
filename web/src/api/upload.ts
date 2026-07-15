import client, { unwrap } from './client'

export async function uploadImage(file: File, subdir = 'products'): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  form.append('subdir', subdir)
  const res = await client.post('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  const data = unwrap<{ url: string }>(res)
  return data.url
}
