# Wiki Clipboard

A simple wiki-style application that serves as a temporary clipboard for text and files.

## Features

- **Text Storage**: Create, view, and edit text entries
- **File Attachments**: Upload and store files with wiki pages
- **Clean UI**: Simple, responsive interface

## Stack

- Backend: Go
- Templates: HTML with CSS styling

## Deployment

### Running Locally

```bash
cd backend
go run wiki.go
```

The application will be available at http://localhost:21313

### Docker Deployment (Vultr)

1. Clone this repository to your Vultr instance
2. Make the deployment script executable:
   ```bash
   cd backend
   chmod +x deploy.sh
   ```
3. Run the deployment script:
   ```bash
   ./deploy.sh
   ```

The application will be available at http://your-vultr-ip:21313

## Usage

- Create/edit a page: Navigate to `/edit/PageName`
- View a page: Navigate to `/view/PageName`
- Upload files: Use the file upload form on any edit page

## API

The application provides a simple JSON API:

- Get page content: `/api/page?title=PageName`
- Uploaded files: `/files/PageName/filename`

## Persistence

Data is stored as text files in the application directory:
- Page content: `PageName.txt`
- File attachments: `files/PageName/filename`
- File lists: `PageName.files.txt`
