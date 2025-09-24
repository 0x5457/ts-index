from sentence_transformers import SentenceTransformer
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn

app = FastAPI()
model = SentenceTransformer('Supabase/gte-small')

class Item(BaseModel):
    sentences: list[str]

@app.post("/embed")
async def embed(item: Item):
    embeddings = model.encode(item.sentences)
    return embeddings.tolist()

@app.get("/health")
async def health():
    return {"status": "ok"}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
