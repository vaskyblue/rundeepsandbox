   POST http://localhost:8000/api/v1/auth/register
   Content-Type: application/json
   
   {
     "username": "rosinver",
     "email": "rosinver.vz@gmail.com",
     "full_name": "RAVD",
     "password": "12345678"
   }

   POST http://localhost:8000/api/v1/auth/token
Content-Type: application/json

{
  "username": "rosinver",
  "password": "123456789"
}

{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDcyNjA4MTQsImlhdCI6MTc0NzI1OTAxNCwic3ViIjoicm9zaW52ZXIifQ.R00_ku5vQ3Z6D4g0nbYtRdk__4djcQ5AW3n_gLKXXA4",
    "token_type": "bearer",
    "expires_in": 1800
}


Voy a buscar las rutas API disponibles en el proyecto para crear peticiones Postman.

Read file: /root/app/app/go-deepsandbox/api.md
Ahora voy a examinar la estructura de las rutas para encontrar todos los endpoints disponibles:

Ran tool

Read file: /root/app/app/go-deepsandbox/routes/routes.go
Ahora voy a revisar los controladores para entender cómo se manejan las solicitudes y qué datos requieren:

Ran tool

Read file: /root/app/app/go-deepsandbox/controllers/auth_controller.go

Read file: /root/app/app/go-deepsandbox/controllers/dataset_controller.go

Read file: /root/app/app/go-deepsandbox/models/user.go

Ran tool

Read file: /root/app/app/go-deepsandbox/models/models.go
Basado en el código revisado, aquí están todas las peticiones para probar la base de datos con Postman:

### 1. Autenticación y Usuarios

#### Registro de Usuario
```
POST http://localhost:8000/api/v1/auth/register
Content-Type: application/json

{
  "username": "usuario_prueba",
  "email": "usuario@ejemplo.com",
  "full_name": "Usuario de Prueba",
  "password": "contraseña123"
}
```

#### Inicio de Sesión
```
POST http://localhost:8000/api/v1/auth/token
Content-Type: application/json

{
  "username": "usuario_prueba",
  "password": "contraseña123"
}
```

#### Obtener Información del Usuario Actual
```
GET http://localhost:8000/api/v1/auth/users/me
Authorization: Bearer {{token}}
```

#### Actualizar Información del Usuario
```
PUT http://localhost:8000/api/v1/auth/users/me
Authorization: Bearer {{token}}
Content-Type: application/json

{
  "email": "nuevo_email@ejemplo.com",
  "full_name": "Nombre Actualizado"
}
```

#### Listar Todos los Usuarios (Solo Admin)
```
GET http://localhost:8000/api/v1/auth/admin/users
Authorization: Bearer {{token}}
```

### 2. Datasets

#### Subir Dataset
```
POST http://localhost:8000/api/v1/datasets/upload
Authorization: Bearer {{token}}
Content-Type: multipart/form-data

file: [seleccionar archivo .csv o .parquet]
```

#### Listar Datasets
```
GET http://localhost:8000/api/v1/datasets
Authorization: Bearer {{token}}
```

#### Obtener Dataset Específico
```
GET http://localhost:8000/api/v1/datasets/{{dataset_id}}
Authorization: Bearer {{token}}
```

#### Eliminar Dataset
```
DELETE http://localhost:8000/api/v1/datasets/{{dataset_id}}
Authorization: Bearer {{token}}
```

### 3. Ejecución de Código

#### Ejecutar Código en Dataset
```
POST http://localhost:8000/api/v1/execute
Authorization: Bearer {{token}}
Content-Type: application/json

{
  "dataset_id": "{{dataset_id}}",
  "code": "print(data.head())",
  "timeout": 300
}
```

#### Obtener Estado de una Tarea
```
GET http://localhost:8000/api/v1/tasks/{{task_id}}
Authorization: Bearer {{token}}
```

#### Cancelar Tarea
```
DELETE http://localhost:8000/api/v1/tasks/{{task_id}}
Authorization: Bearer {{token}}
```

#### Obtener Estado de la Cola (Solo Admin)
```
GET http://localhost:8000/api/v1/admin/queue-status
Authorization: Bearer {{token}}
```

Para usar estas peticiones en Postman:
1. Crea una nueva colección
2. Configura una variable de entorno llamada `token` que se actualizará después del login
3. Primero registra un usuario, luego inicia sesión y guarda el token recibido
4. Usa ese token para las demás peticiones que requieren autenticación
