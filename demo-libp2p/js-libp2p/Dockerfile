FROM node:21.5.0-alpine3.19

WORKDIR /app
COPY . /app/

RUN npm install
    
EXPOSE 3000

CMD ["node", "src/index.js"]