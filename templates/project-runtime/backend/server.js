const http = require('node:http')

http.createServer((request, response) => {
  response.setHeader('content-type', 'application/json; charset=utf-8')
  if (request.url === '/hello') {
    response.end(JSON.stringify({ message: '前后端已经在本站连通。' }))
    return
  }
  response.statusCode = 404
  response.end(JSON.stringify({ error: 'not found' }))
}).listen(8080, '0.0.0.0')
