import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { RegisterComponent } from './register';
import { AuthService } from '../../../core/services/auth';
import { CaptchaService } from '../../../core/captcha/captcha.service';

describe('RegisterComponent', () => {
  let component: RegisterComponent;
  let fixture: ComponentFixture<RegisterComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RegisterComponent],
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            register: () => of({}),
          },
        },
        {
          provide: CaptchaService,
          useValue: {
            getConfig: () => of({ enabled: false, site_key: '' }),
            loadScript: () => Promise.resolve(),
            render: () => undefined,
            remove: () => undefined,
            reset: () => undefined,
            clearCache: () => undefined,
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RegisterComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
