describe("Admin Panel — Time Request Approval Affects User Quota", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("increases the user's qpu_seconds when a time request is approved", () => {
    // 1. Log in as regular user and note current quota
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Settings").click();
    cy.contains("h1", "Settings").should("be.visible");

    cy.contains("QPU Balance")
      .parent()
      .within(() => {
        cy.get("span.font-mono")
          .invoke("text")
          .then((text) => parseFloat(text.replace("s", "").trim()))
          .as("initialQuota");
      });

    // 2. Submit a time request from the sidebar
    cy.contains("button", "Request Time").click();
    cy.contains("h3", "Request QPU Time").should("be.visible");

    cy.get('input[type="number"]').type("{selectall}{backspace}300");
    cy.get('textarea[placeholder="Running VQE experiments for chemistry simulations..."]')
      .clear().type("Need extra time for testing");
      
    const alertStub = cy.stub();
    cy.on("window:alert", alertStub);
    cy.contains("button", "Submit Time Request").click();
    cy.wrap(alertStub).should("have.been.calledWithMatch", /submitted successfully/);

    // 3. Log out and log in as admin
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // 4. Navigate to Admin Panel → Time Requests and approve
    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Time Requests").click();

    cy.get('[data-testid="time-request-row"]').first().as('requestRow');
    cy.get('@requestRow').within(() => {
      cy.get('button svg.lucide-check').parent().click();
    });

    // Verify the request is now approved
    cy.get('@requestRow').within(() => {
      cy.contains("span", "approved").should("be.visible");
    });

    // 5. Log out and log back in as the regular user
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // 6. Check Settings — quota should have increased by 300
    cy.contains("button", "Settings").click();
    cy.contains("h1", "Settings").should("be.visible");

    cy.contains("QPU Balance")
      .parent()
      .within(() => {
        cy.get("span.font-mono")
          .invoke("text")
          .then((text) => parseFloat(text.replace("s", "").trim()))
          .then((newQuota) => {
            cy.get("@initialQuota").then((initialQuota) => {
              expect(newQuota).to.equal(Number(initialQuota) + 300);
            });
          });
      });
  });

  it("does not change the user's qpu_seconds when a time request is rejected", () => {
    // 1. Log in as regular user and note current quota
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Settings").click();
    cy.contains("h1", "Settings").should("be.visible");

    cy.contains("QPU Balance")
      .parent()
      .within(() => {
        cy.get("span.font-mono")
          .invoke("text")
          .then((text) => parseFloat(text.replace("s", "").trim()))
          .as("initialQuota");
      });

    // 2. Submit a time request
    cy.contains("button", "Request Time").click();
    cy.contains("h3", "Request QPU Time").should("be.visible");

    cy.get('input[type="number"]').type("{selectall}{backspace}200");
    cy.get('textarea[placeholder="Running VQE experiments for chemistry simulations..."]')
      .clear().type("Testing rejection flow");
      
    const alertStub = cy.stub();
    cy.on("window:alert", alertStub);
    cy.contains("button", "Submit Time Request").click();
    cy.wrap(alertStub).should("have.been.calledWithMatch", /submitted successfully/);

    // 3. Log out and log in as admin
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // 4. Navigate to Admin Panel → Time Requests and reject
    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Time Requests").click();

    cy.window().then((win) => {
      cy.stub(win, "prompt").returns("Not enough justification");
    });

    cy.get('[data-testid="time-request-row"]').first().as('requestRow');
    cy.get('@requestRow').within(() => {
      cy.get('button svg.lucide-x').parent().click();
    });

    // Verify the request is now rejected
    cy.get('@requestRow').within(() => {
      cy.contains("span", "rejected").should("be.visible");
    });

    // 5. Log out and log back in as the regular user
    cy.contains("button", "Profile Settings").click();
    cy.contains("button", "Sign Out").click();
    cy.get('[data-testid="login-modal"]').should("be.visible");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // 6. Check Settings — quota should remain unchanged
    cy.contains("button", "Settings").click();
    cy.contains("h1", "Settings").should("be.visible");

    cy.contains("QPU Balance")
      .parent()
      .within(() => {
        cy.get("span.font-mono")
          .invoke("text")
          .then((text) => parseFloat(text.replace("s", "").trim()))
          .then((newQuota) => {
            cy.get("@initialQuota").then((initialQuota) => {
              expect(newQuota).to.equal(Number(initialQuota));
            });
          });
      });
  });
});
