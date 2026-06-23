import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { FormsModule } from '@angular/forms';

interface APIKey {
  id: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string;
}

interface CreateAPIKeyResponse {
  id: string;
  name: string;
  prefix: string;
  raw_key: string;
  created_at: string;
}

@Component({
  selector: 'app-api-keys',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './api-keys.html',
  styleUrl: './api-keys.css'
})
export class ApiKeysComponent implements OnInit {
  private http = inject(HttpClient);

  keys = signal<APIKey[]>([]);
  showCreateModal = signal(false);
  showSuccessModal = signal(false);
  newKeyName = signal('');
  createdKey = signal<CreateAPIKeyResponse | null>(null);

  ngOnInit() {
    this.loadKeys();
  }

  loadKeys() {
    this.http.get<APIKey[]>('/api/v1/api-keys').subscribe(keys => {
      this.keys.set(keys || []);
    });
  }

  openCreateModal() {
    this.newKeyName.set('');
    this.showCreateModal.set(true);
  }

  closeCreateModal() {
    this.showCreateModal.set(false);
  }

  createKey() {
    if (!this.newKeyName()) return;

    this.http.post<CreateAPIKeyResponse>('/api/v1/api-keys', { name: this.newKeyName() })
      .subscribe(res => {
        this.createdKey.set(res);
        this.showCreateModal.set(false);
        this.showSuccessModal.set(true);
        this.loadKeys();
      });
  }

  closeSuccessModal() {
    this.showSuccessModal.set(false);
    this.createdKey.set(null);
  }

  revokeKey(id: string) {
    if (!confirm('Are you sure you want to revoke this API key? This action cannot be undone.')) return;

    this.http.delete(`/api/v1/api-keys/${id}`).subscribe(() => {
      this.loadKeys();
    });
  }

  copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
  }
}
