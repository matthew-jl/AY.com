<script lang="ts">
    import { onMount } from 'svelte';
    import { 
      Users, 
      UserX, 
      UserCheck,
      BadgeCheck,
      Flag,
      Send,
      Building2,
      CheckCircle,
      XCircle,
      Tag,
      Plus,
      Pencil,
      Trash2,
      Mail,
      Filter,
      Search,
      AlertTriangle,
      Crown,
      RefreshCcw,
      Hash,
      BellRing
    } from 'lucide-svelte';
  
    // Tab definitions
    type AdminTab = 'users' | 'communities' | 'reports' | 'premium' | 'categories' | 'newsletter';
    let activeTab: AdminTab = 'users';
    
    // Loading states
    let loadingStates = {
      users: true,
      communities: false,
      reports: false,
      premium: false,
      categories: false,
      threadCategories: false,
      communityCategories: false,
      newsletter: false,
      banAction: false,
      unbanAction: false,
      categoryAction: false,
      newsletterAction: false,
      communityRequestAction: false,
      premiumRequestAction: false,
      reportRequestAction: false
    };
  
    // Error states
    let errors: {
      users: string | null,
      communities: string | null,
      reports: string | null,
      premium: string | null,
      categories: string | null,
      newsletter: string | null,
      action: string | null
    } = {
      users: null,
      communities: null,
      reports: null,
      premium: null,
      categories: null,
      newsletter: null,
      action: null
    };
    
    // Data models
    interface User {
      id: number;
      username: string;
      name: string;
      email: string;
      profile_picture: string | null;
      is_banned: boolean;
      account_status: string;
      is_premium: boolean;
      created_at: string;
      subscribed_to_newsletter: boolean;
    }
    
    interface CommunityRequest {
      id: number;
      name: string;
      description: string;
      creator: {
        id: number;
        email: string;
        name: string;
        username: string;
        profile_picture: string | null;
      };
      created_at: string;
      status: 'pending' | 'approved' | 'rejected';
    }
    
    interface PremiumRequest {
      id: number;
      user: {
        id: number;
        name: string;
        username: string;
        profile_picture: string | null;
      };
      created_at: string;
      status: 'pending' | 'approved' | 'rejected';
      payment_proof_url: string;
    }
    
    interface ReportRequest {
      id: number;
      reporter: {
        id: number;
        name: string;
        username: string;
      };
      reported_entity_type: 'thread' | 'user' | 'community';
      reported_entity_id: number;
      reported_entity_name: string;
      reason: string;
      created_at: string;
      status: 'pending' | 'resolved' | 'dismissed';
    }
    
    interface Category {
      id: number;
      name: string;
      description: string;
      created_at: string;
      count: number; // usage count
    }
    
    // Data state
    let users: User[] = [];
    let filteredUsers: User[] = [];
    let communityRequests: CommunityRequest[] = [];
    let premiumRequests: PremiumRequest[] = [];
    let reportRequests: ReportRequest[] = [];
    let threadCategories: Category[] = [];
    let communityCategories: Category[] = [];
    
    // Filters and search
    let userSearchQuery = '';
    let userStatusFilter = 'all';
    let requestFilter = 'pending';
    
    // Form states
    let showNewsletter = false;
    let newsletterSubject = '';
    let newsletterBody = '';
    let newsletterSending = false;
    
    let editingThreadCategory: Category | null = null;
    let newThreadCategory = { name: '', description: '' };
    
    let editingCommunityCategory: Category | null = null;
    let newCommunityCategory = { name: '', description: '' };
    
    let viewingImage: string | null = null;
    
    // Lifecycle
    onMount(async () => {
      await loadUsers();
    });
    
    // Tab navigation
    function setActiveTab(tab: AdminTab) {
      if (activeTab === tab) return;
      
      activeTab = tab;
      errors.action = null;
      
      // Load data based on active tab
      if (tab === 'users' && users.length === 0) {
        loadUsers();
      } else if (tab === 'communities' && communityRequests.length === 0) {
        loadCommunityRequests();
      } else if (tab === 'reports' && reportRequests.length === 0) {
        loadReportRequests();
      } else if (tab === 'premium' && premiumRequests.length === 0) {
        loadPremiumRequests();
      } else if (tab === 'categories') {
        if (threadCategories.length === 0) loadThreadCategories();
        if (communityCategories.length === 0) loadCommunityCategories();
      }
    }
  
    // Data loading functions
    async function loadUsers() {
      loadingStates.users = true;
      errors.users = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 800));
        
        // Dummy data
        users = [
          {
            id: 1,
            username: 'johndoe',
            name: 'John Doe',
            email: 'john@example.com',
            profile_picture: 'https://i.pravatar.cc/150?img=1',
            is_banned: false,
            account_status: 'active',
            is_premium: true,
            created_at: '2025-01-15T08:30:00Z',
            subscribed_to_newsletter: true
          },
          {
            id: 2,
            username: 'janedoe',
            name: 'Jane Doe',
            email: 'jane@example.com',
            profile_picture: 'https://i.pravatar.cc/150?img=5',
            is_banned: false,
            account_status: 'active',
            is_premium: false,
            created_at: '2025-02-20T14:15:00Z',
            subscribed_to_newsletter: true
          },
          {
            id: 3,
            username: 'banneduser',
            name: 'Banned User',
            email: 'banned@example.com',
            profile_picture: null,
            is_banned: true,
            account_status: 'banned',
            is_premium: false,
            created_at: '2025-03-10T11:45:00Z',
            subscribed_to_newsletter: false
          }
        ];
        
        applyUserFilters();
      } catch (error) {
        console.error('Failed to load users:', error);
        errors.users = 'Failed to load users. Please try again.';
      } finally {
        loadingStates.users = false;
      }
    }
    
    async function loadCommunityRequests() {
      loadingStates.communities = true;
      errors.communities = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 600));
        
        // Dummy data
        communityRequests = [
          {
            id: 101,
            name: 'Photography Enthusiasts',
            description: 'A community for sharing photography tips and showcasing your best shots.',
            creator: {
              id: 1,
              email: 'johndoe@gmail.com',
              name: 'John Doe',
              username: 'johndoe',
              profile_picture: 'https://i.pravatar.cc/150?img=1'
            },
            created_at: '2025-06-10T09:20:00Z',
            status: 'pending'
          },
          {
            id: 102,
            name: 'Web Development',
            description: 'Discuss the latest in web development technologies and frameworks.',
            creator: {
              id: 2,
              email: 'janedoe@gmail.com',
              name: 'Jane Doe',
              username: 'janedoe',
              profile_picture: 'https://i.pravatar.cc/150?img=5'
            },
            created_at: '2025-06-12T15:45:00Z',
            status: 'pending'
          }
        ];
      } catch (error) {
        console.error('Failed to load community requests:', error);
        errors.communities = 'Failed to load community requests. Please try again.';
      } finally {
        loadingStates.communities = false;
      }
    }
    
    async function loadPremiumRequests() {
      loadingStates.premium = true;
      errors.premium = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 700));
        
        // Dummy data
        premiumRequests = [
          {
            id: 201,
            user: {
              id: 2,
              name: 'Jane Doe',
              username: 'janedoe',
              profile_picture: 'https://i.pravatar.cc/150?img=5'
            },
            created_at: '2025-06-09T10:30:00Z',
            status: 'pending',
            payment_proof_url: 'https://placehold.co/600x400/png?text=Payment+Receipt'
          },
          {
            id: 202,
            user: {
              id: 4,
              name: 'Alex Smith',
              username: 'alexsmith',
              profile_picture: 'https://i.pravatar.cc/150?img=8'
            },
            created_at: '2025-06-11T16:15:00Z',
            status: 'pending',
            payment_proof_url: 'https://placehold.co/600x400/png?text=Payment+Receipt'
          }
        ];
      } catch (error) {
        console.error('Failed to load premium requests:', error);
        errors.premium = 'Failed to load premium requests. Please try again.';
      } finally {
        loadingStates.premium = false;
      }
    }
    
    async function loadReportRequests() {
      loadingStates.reports = true;
      errors.reports = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Dummy data
        reportRequests = [
          {
            id: 301,
            reporter: {
              id: 1,
              name: 'John Doe',
              username: 'johndoe'
            },
            reported_entity_type: 'thread',
            reported_entity_id: 1001,
            reported_entity_name: 'Thread with inappropriate content',
            reason: 'This thread contains misleading information and spam links.',
            created_at: '2025-06-10T08:45:00Z',
            status: 'pending'
          },
          {
            id: 302,
            reporter: {
              id: 2,
              name: 'Jane Doe',
              username: 'janedoe'
            },
            reported_entity_type: 'user',
            reported_entity_id: 5,
            reported_entity_name: 'spamuser123',
            reason: 'This user is sending unsolicited promotional messages.',
            created_at: '2025-06-11T14:20:00Z',
            status: 'pending'
          },
          {
            id: 303,
            reporter: {
              id: 4,
              name: 'Alex Smith',
              username: 'alexsmith'
            },
            reported_entity_type: 'community',
            reported_entity_id: 15,
            reported_entity_name: 'Controversial Topics',
            reason: 'This community regularly hosts discussions that violate platform guidelines.',
            created_at: '2025-06-12T11:10:00Z',
            status: 'pending'
          }
        ];
      } catch (error) {
        console.error('Failed to load report requests:', error);
        errors.reports = 'Failed to load report requests. Please try again.';
      } finally {
        loadingStates.reports = false;
      }
    }
    
    async function loadThreadCategories() {
      loadingStates.threadCategories = true;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 400));
        
        // Dummy data
        threadCategories = [
          {
            id: 1,
            name: 'Discussion',
            description: 'General discussion topics',
            created_at: '2024-12-01T00:00:00Z',
            count: 145
          },
          {
            id: 2,
            name: 'Question',
            description: 'Questions seeking answers or help',
            created_at: '2024-12-01T00:00:00Z',
            count: 87
          },
          {
            id: 3,
            name: 'Announcement',
            description: 'Official announcements and news',
            created_at: '2024-12-01T00:00:00Z',
            count: 32
          },
          {
            id: 4,
            name: 'Showcase',
            description: 'Share your work and achievements',
            created_at: '2025-01-15T00:00:00Z',
            count: 56
          }
        ];
      } catch (error) {
        console.error('Failed to load thread categories:', error);
        errors.categories = 'Failed to load thread categories. Please try again.';
      } finally {
        loadingStates.threadCategories = false;
      }
    }
    
    async function loadCommunityCategories() {
      loadingStates.communityCategories = true;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 300));
        
        // Dummy data
        communityCategories = [
          {
            id: 101,
            name: 'Technology',
            description: 'Technology related communities',
            created_at: '2024-12-01T00:00:00Z',
            count: 18
          },
          {
            id: 102,
            name: 'Arts',
            description: 'Arts and creative communities',
            created_at: '2024-12-01T00:00:00Z',
            count: 12
          },
          {
            id: 103,
            name: 'Science',
            description: 'Science focused communities',
            created_at: '2024-12-01T00:00:00Z',
            count: 9
          },
          {
            id: 104,
            name: 'Entertainment',
            description: 'Entertainment and media communities',
            created_at: '2025-01-20T00:00:00Z',
            count: 21
          }
        ];
      } catch (error) {
        console.error('Failed to load community categories:', error);
        errors.categories = 'Failed to load community categories. Please try again.';
      } finally {
        loadingStates.communityCategories = false;
      }
    }
    
    // Filter functions
    function applyUserFilters() {
      filteredUsers = users.filter(user => {
        // Status filter
        if (userStatusFilter !== 'all') {
          if (userStatusFilter === 'banned' && !user.is_banned) return false;
          if (userStatusFilter === 'active' && user.is_banned) return false;
          if (userStatusFilter === 'premium' && !user.is_premium) return false;
          if (userStatusFilter === 'newsletter' && !user.subscribed_to_newsletter) return false;
        }
        
        // Search query
        if (userSearchQuery) {
          const query = userSearchQuery.toLowerCase();
          return (
            user.name.toLowerCase().includes(query) ||
            user.username.toLowerCase().includes(query) ||
            user.email.toLowerCase().includes(query)
          );
        }
        
        return true;
      });
    }
    
    $: if (userSearchQuery !== undefined || userStatusFilter) {
      applyUserFilters();
    }
    
    // Action functions
    async function banUser(user: User) {
      loadingStates.banAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 600));
        
        // Update local state
        user.is_banned = true;
        user.account_status = 'banned';
        
        applyUserFilters();
        alert(`User ${user.username} has been banned.`);
      } catch (error) {
        console.error('Failed to ban user:', error);
        errors.action = 'Failed to ban user. Please try again.';
      } finally {
        loadingStates.banAction = false;
      }
    }
    
    async function unbanUser(user: User) {
      loadingStates.unbanAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 600));
        
        // Update local state
        user.is_banned = false;
        user.account_status = 'active';
        
        applyUserFilters();
        alert(`User ${user.username} has been unbanned.`);
      } catch (error) {
        console.error('Failed to unban user:', error);
        errors.action = 'Failed to unban user. Please try again.';
      } finally {
        loadingStates.unbanAction = false;
      }
    }
    
    async function handleCommunityRequest(request: CommunityRequest, approve: boolean) {
      loadingStates.communityRequestAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 800));
        
        // Update local state
        request.status = approve ? 'approved' : 'rejected';
        
        alert(`Community request "${request.name}" has been ${approve ? 'approved' : 'rejected'}.`);
        
        if (approve) {
          // Simulate email sending
          alert(`Email notification sent to ${request.creator.email || request.creator.username} about community approval.`);
        }
      } catch (error) {
        console.error('Failed to process community request:', error);
        errors.action = 'Failed to process community request. Please try again.';
      } finally {
        loadingStates.communityRequestAction = false;
      }
    }
    
    async function handlePremiumRequest(request: PremiumRequest, approve: boolean) {
      loadingStates.premiumRequestAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 700));
        
        // Update local state
        request.status = approve ? 'approved' : 'rejected';
        
        if (approve) {
          // Update user premium status if it exists in our users array
          const user = users.find(u => u.id === request.user.id);
          if (user) {
            user.is_premium = true;
            applyUserFilters();
          }
          
          // Simulate email sending
          alert(`Email notification sent to ${request.user.username} about premium approval.`);
        }
        
        alert(`Premium request for ${request.user.username} has been ${approve ? 'approved' : 'rejected'}.`);
      } catch (error) {
        console.error('Failed to process premium request:', error);
        errors.action = 'Failed to process premium request. Please try again.';
      } finally {
        loadingStates.premiumRequestAction = false;
      }
    }
    
    async function handleReportRequest(report: ReportRequest, approve: boolean) {
      loadingStates.reportRequestAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 600));
        
        // Update local state
        report.status = approve ? 'resolved' : 'dismissed';
        
        alert(`Report request #${report.id} has been ${approve ? 'resolved' : 'dismissed'}.`);
        
        if (approve) {
          // Simulate email sending to reporter
          alert(`Email notification sent to ${report.reporter.username} about report resolution.`);
        }
      } catch (error) {
        console.error('Failed to process report request:', error);
        errors.action = 'Failed to process report request. Please try again.';
      } finally {
        loadingStates.reportRequestAction = false;
      }
    }
    
    // Category management
    function startEditThreadCategory(category: Category | null = null) {
      editingThreadCategory = category ? { ...category } : null;
      
      if (!category) {
        newThreadCategory = { name: '', description: '' };
      }
    }
    
    function startEditCommunityCategory(category: Category | null = null) {
      editingCommunityCategory = category ? { ...category } : null;
      
      if (!category) {
        newCommunityCategory = { name: '', description: '' };
      }
    }
    
    async function saveThreadCategory() {
      loadingStates.categoryAction = true;
      errors.action = null;
      
      try {
        // Validate
        if (!newThreadCategory.name.trim()) {
          errors.action = 'Category name is required';
          return;
        }
        
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 500));
        
        if (editingThreadCategory !== null) {
          // Update existing category
          const index = threadCategories.findIndex(c => c.id === editingThreadCategory!.id);
          if (index !== -1) {
            threadCategories[index] = {
              ...threadCategories[index],
              name: newThreadCategory.name,
              description: newThreadCategory.description
            };
            alert(`Thread category "${newThreadCategory.name}" has been updated.`);
          }
        } else {
          // Create new category
          const newId = Math.max(...threadCategories.map(c => c.id), 0) + 1;
          threadCategories = [
            ...threadCategories,
            {
              id: newId,
              name: newThreadCategory.name,
              description: newThreadCategory.description,
              created_at: new Date().toISOString(),
              count: 0
            }
          ];
          alert(`Thread category "${newThreadCategory.name}" has been created.`);
        }
        
        // Reset form
        editingThreadCategory = null;
        newThreadCategory = { name: '', description: '' };
      } catch (error) {
        console.error('Failed to save thread category:', error);
        errors.action = 'Failed to save thread category. Please try again.';
      } finally {
        loadingStates.categoryAction = false;
      }
    }
    
    async function saveCommunityCategory() {
      loadingStates.categoryAction = true;
      errors.action = null;
      
      try {
        // Validate
        if (!newCommunityCategory.name.trim()) {
          errors.action = 'Category name is required';
          return;
        }
        
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 500));
        
        if (editingCommunityCategory) {
            let tempId = editingCommunityCategory.id;
          // Update existing category
          const index = communityCategories.findIndex(c => c.id === tempId);
          if (index !== -1) {
            communityCategories[index] = {
              ...communityCategories[index],
              name: newCommunityCategory.name,
              description: newCommunityCategory.description
            };
            alert(`Community category "${newCommunityCategory.name}" has been updated.`);
          }
        } else {
          // Create new category
          const newId = Math.max(...communityCategories.map(c => c.id), 100) + 1;
          communityCategories = [
            ...communityCategories,
            {
              id: newId,
              name: newCommunityCategory.name,
              description: newCommunityCategory.description,
              created_at: new Date().toISOString(),
              count: 0
            }
          ];
          alert(`Community category "${newCommunityCategory.name}" has been created.`);
        }
        
        // Reset form
        editingCommunityCategory = null;
        newCommunityCategory = { name: '', description: '' };
      } catch (error) {
        console.error('Failed to save community category:', error);
        errors.action = 'Failed to save community category. Please try again.';
      } finally {
        loadingStates.categoryAction = false;
      }
    }
    
    async function deleteThreadCategory(category: Category) {
      if (!confirm(`Are you sure you want to delete category "${category.name}"?`)) {
        return;
      }
      
      loadingStates.categoryAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Update local state
        threadCategories = threadCategories.filter(c => c.id !== category.id);
        
        alert(`Thread category "${category.name}" has been deleted.`);
      } catch (error) {
        console.error('Failed to delete thread category:', error);
        errors.action = 'Failed to delete thread category. Please try again.';
      } finally {
        loadingStates.categoryAction = false;
      }
    }
    
    async function deleteCommunityCategory(category: Category) {
      if (!confirm(`Are you sure you want to delete category "${category.name}"?`)) {
        return;
      }
      
      loadingStates.categoryAction = true;
      errors.action = null;
      
      try {
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Update local state
        communityCategories = communityCategories.filter(c => c.id !== category.id);
        
        alert(`Community category "${category.name}" has been deleted.`);
      } catch (error) {
        console.error('Failed to delete community category:', error);
        errors.action = 'Failed to delete community category. Please try again.';
      } finally {
        loadingStates.categoryAction = false;
      }
    }
    
    // Newsletter
    function toggleNewsletterForm() {
      showNewsletter = !showNewsletter;
      if (!showNewsletter) {
        newsletterSubject = '';
        newsletterBody = '';
      }
    }
    
    async function sendNewsletter() {
      if (!newsletterSubject.trim() || !newsletterBody.trim()) {
        errors.action = 'Please provide both subject and body for the newsletter.';
        return;
      }
      
      loadingStates.newsletterAction = true;
      errors.action = null;
      
      try {
        // Count subscribers
        const subscriberCount = users.filter(user => user.subscribed_to_newsletter).length;
        
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        alert(`Newsletter sent to ${subscriberCount} subscribers!`);
        
        // Reset form
        showNewsletter = false;
        newsletterSubject = '';
        newsletterBody = '';
      } catch (error) {
        console.error('Failed to send newsletter:', error);
        errors.action = 'Failed to send newsletter. Please try again.';
      } finally {
        loadingStates.newsletterAction = false;
      }
    }
    
    function formatDate(dateString: string) {
      const date = new Date(dateString);
      return new Intl.DateTimeFormat('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      }).format(date);
    }
  </script>
  
  <div class="admin-page">
    <header class="admin-header">
      <h1>Admin Dashboard</h1>
    </header>
  
    <div class="admin-container">
      <!-- Tabs navigation -->
      <div class="admin-tabs">
        <button 
          class="tab-button" 
          class:active={activeTab === 'users'} 
          on:click={() => setActiveTab('users')}
        >
          <Users size={18} />
          <span class="tab-text">Users</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'communities'} 
          on:click={() => setActiveTab('communities')}
        >
          <Building2 size={18} />
          <span class="tab-text">Community Requests</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'premium'} 
          on:click={() => setActiveTab('premium')}
        >
          <Crown size={18} />
          <span class="tab-text">Premium Requests</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'reports'} 
          on:click={() => setActiveTab('reports')}
        >
          <Flag size={18} />
          <span class="tab-text">Reports</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'categories'} 
          on:click={() => setActiveTab('categories')}
        >
          <Tag size={18} />
          <span class="tab-text">Categories</span>
        </button>
        
        <button 
          class="tab-button" 
          class:active={activeTab === 'newsletter'} 
          on:click={() => setActiveTab('newsletter')}
        >
          <Mail size={18} />
          <span class="tab-text">Newsletter</span>
        </button>
      </div>
  
      <!-- Tab content -->
      <div class="tab-content">
        <!-- Users Tab -->
        {#if activeTab === 'users'}
          <div class="tab-panel">
            <div class="panel-header">
              <h2>
                <Users size={22} />
                <span>Manage Users</span>
              </h2>
              
              <div class="filter-controls">
                <div class="search-box">
                  <Search size={18} />
                  <input 
                    type="text" 
                    placeholder="Search users..." 
                    bind:value={userSearchQuery} 
                    on:input={applyUserFilters}
                  />
                </div>
                
                <div class="filter-dropdown">
                  <Filter size={18} />
                  <select bind:value={userStatusFilter} on:change={applyUserFilters}>
                    <option value="all">All Users</option>
                    <option value="active">Active</option>
                    <option value="banned">Banned</option>
                    <option value="premium">Premium</option>
                    <option value="newsletter">Newsletter Subscribers</option>
                  </select>
                </div>
                
                <button class="btn btn-outline btn-icon" on:click={loadUsers}>
                  <RefreshCcw size={18} />
                  <span>Refresh</span>
                </button>
              </div>
            </div>
            
            {#if loadingStates.users}
              <div class="loading-container">
                <div class="loading-spinner"></div>
                <p>Loading users...</p>
              </div>
            {:else if errors.users}
              <div class="error-message">
                <AlertTriangle size={20} />
                <p>{errors.users}</p>
                <button class="btn btn-primary" on:click={loadUsers}>Try Again</button>
              </div>
            {:else if filteredUsers.length === 0}
              <div class="empty-state">
                <Users size={40} />
                <p>No users found matching your criteria</p>
                {#if userSearchQuery || userStatusFilter !== 'all'}
                  <button class="btn btn-outline" on:click={() => { userSearchQuery = ''; userStatusFilter = 'all'; }}>
                    Clear Filters
                  </button>
                {/if}
              </div>
            {:else}
              <div class="user-list">
                <table class="data-table">
                  <thead>
                    <tr>
                      <th>User</th>
                      <th class="hide-sm">Email</th>
                      <th class="hide-sm">Created</th>
                      <th>Status</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each filteredUsers as user (user.id)}
                      <tr class:banned={user.is_banned}>
                        <td class="user-cell">
                          <div class="user-info">
                            <div class="avatar">
                              {#if user.profile_picture}
                                <img src={user.profile_picture} alt={user.name} />
                              {:else}
                                <div class="avatar-placeholder">{user.name[0]}</div>
                              {/if}
                            </div>
                            <div class="user-details">
                              <span class="user-name">{user.name}</span>
                              <span class="user-username">@{user.username}</span>
                            </div>
                          </div>
                        </td>
                        <td class="hide-sm">{user.email}</td>
                        <td class="hide-sm">{formatDate(user.created_at)}</td>
                        <td>
                          <div class="status-badges">
                            {#if user.is_banned}
                              <span class="badge badge-danger">Banned</span>
                            {:else}
                              <span class="badge badge-success">Active</span>
                            {/if}
                            
                            {#if user.is_premium}
                              <span class="badge badge-premium">Premium</span>
                            {/if}
                            
                            {#if user.subscribed_to_newsletter}
                              <span class="badge badge-info hide-sm">Newsletter</span>
                            {/if}
                          </div>
                        </td>
                        <td>
                          <div class="action-buttons">
                            {#if user.is_banned}
                              <button 
                                class="btn btn-success btn-sm" 
                                on:click={() => unbanUser(user)}
                                disabled={loadingStates.unbanAction}
                              >
                                <UserCheck size={16} />
                                <span class="hide-sm">Unban</span>
                              </button>
                            {:else}
                              <button 
                                class="btn btn-danger btn-sm" 
                                on:click={() => banUser(user)}
                                disabled={loadingStates.banAction}
                              >
                                <UserX size={16} />
                                <span class="hide-sm">Ban</span>
                              </button>
                            {/if}
                          </div>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>
        {/if}
        
        <!-- Community Requests Tab -->
        {#if activeTab === 'communities'}
          <div class="tab-panel">
            <div class="panel-header">
              <h2>
                <Building2 size={22} />
                <span>Community Creation Requests</span>
              </h2>
              
              <div class="filter-controls">
                <div class="filter-dropdown">
                  <Filter size={18} />
                  <select bind:value={requestFilter}>
                    <option value="pending">Pending</option>
                    <option value="approved">Approved</option>
                    <option value="rejected">Rejected</option>
                    <option value="all">All Requests</option>
                  </select>
                </div>
                
                <button class="btn btn-outline btn-icon" on:click={loadCommunityRequests}>
                  <RefreshCcw size={18} />
                  <span>Refresh</span>
                </button>
              </div>
            </div>
            
            {#if loadingStates.communities}
              <div class="loading-container">
                <div class="loading-spinner"></div>
                <p>Loading community requests...</p>
              </div>
            {:else if errors.communities}
              <div class="error-message">
                <AlertTriangle size={20} />
                <p>{errors.communities}</p>
                <button class="btn btn-primary" on:click={loadCommunityRequests}>Try Again</button>
              </div>
            {:else if communityRequests.length === 0 || !communityRequests.filter(r => requestFilter === 'all' || r.status === requestFilter).length}
              <div class="empty-state">
                <Building2 size={40} />
                <p>No community requests {requestFilter !== 'all' ? `with status "${requestFilter}"` : ''}</p>
                {#if requestFilter !== 'pending'}
                  <button class="btn btn-outline" on:click={() => requestFilter = 'pending'}>
                    Show Pending Requests
                  </button>
                {/if}
              </div>
            {:else}
              <div class="request-list">
                {#each communityRequests.filter(r => requestFilter === 'all' || r.status === requestFilter) as request (request.id)}
                  <div class="request-card" class:resolved={request.status !== 'pending'}>
                    <div class="request-header">
                      <h3>{request.name}</h3>
                      <div class="request-meta">
                        <span>Requested {formatDate(request.created_at)}</span>
                        
                        {#if request.status !== 'pending'}
                          <span class="badge badge-{request.status === 'approved' ? 'success' : 'danger'}">
                            {request.status}
                          </span>
                        {/if}
                      </div>
                    </div>
                    
                    <div class="request-body">
                      <div class="request-creator">
                        <div class="avatar small">
                          {#if request.creator.profile_picture}
                            <img src={request.creator.profile_picture} alt={request.creator.name} />
                          {:else}
                            <div class="avatar-placeholder">{request.creator.name[0]}</div>
                          {/if}
                        </div>
                        <span>By <strong>{request.creator.name}</strong> (@{request.creator.username})</span>
                      </div>
                      
                      <p class="request-description">{request.description}</p>
                    </div>
                    
                    {#if request.status === 'pending'}
                      <div class="request-actions">
                        <button 
                          class="btn btn-danger" 
                          on:click={() => handleCommunityRequest(request, false)}
                          disabled={loadingStates.communityRequestAction}
                        >
                          <XCircle size={18} />
                          Reject
                        </button>
                        <button 
                          class="btn btn-success" 
                          on:click={() => handleCommunityRequest(request, true)}
                          disabled={loadingStates.communityRequestAction}
                        >
                          <CheckCircle size={18} />
                          Approve
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
        
        <!-- Premium Requests Tab -->
        {#if activeTab === 'premium'}
          <div class="tab-panel">
            <div class="panel-header">
              <h2>
                <Crown size={22} />
                <span>Premium Requests</span>
              </h2>
              
              <div class="filter-controls">
                <div class="filter-dropdown">
                  <Filter size={18} />
                  <select bind:value={requestFilter}>
                    <option value="pending">Pending</option>
                    <option value="approved">Approved</option>
                    <option value="rejected">Rejected</option>
                    <option value="all">All Requests</option>
                  </select>
                </div>
                
                <button class="btn btn-outline btn-icon" on:click={loadPremiumRequests}>
                  <RefreshCcw size={18} />
                  <span>Refresh</span>
                </button>
              </div>
            </div>
            
            {#if loadingStates.premium}
              <div class="loading-container">
                <div class="loading-spinner"></div>
                <p>Loading premium requests...</p>
              </div>
            {:else if errors.premium}
              <div class="error-message">
                <AlertTriangle size={20} />
                <p>{errors.premium}</p>
                <button class="btn btn-primary" on:click={loadPremiumRequests}>Try Again</button>
              </div>
            {:else if premiumRequests.length === 0 || !premiumRequests.filter(r => requestFilter === 'all' || r.status === requestFilter).length}
              <div class="empty-state">
                <Crown size={40} />
                <p>No premium requests {requestFilter !== 'all' ? `with status "${requestFilter}"` : ''}</p>
                {#if requestFilter !== 'pending'}
                  <button class="btn btn-outline" on:click={() => requestFilter = 'pending'}>
                    Show Pending Requests
                  </button>
                {/if}
              </div>
            {:else}
              <div class="request-list">
                {#each premiumRequests.filter(r => requestFilter === 'all' || r.status === requestFilter) as request (request.id)}
                  <div class="request-card" class:resolved={request.status !== 'pending'}>
                    <div class="request-header">
                      <h3>Premium Request: {request.user.name}</h3>
                      <div class="request-meta">
                        <span>Requested {formatDate(request.created_at)}</span>
                        
                        {#if request.status !== 'pending'}
                          <span class="badge badge-{request.status === 'approved' ? 'success' : 'danger'}">
                            {request.status}
                          </span>
                        {/if}
                      </div>
                    </div>
                    
                    <div class="request-body">
                      <div class="request-creator">
                        <div class="avatar small">
                          {#if request.user.profile_picture}
                            <img src={request.user.profile_picture} alt={request.user.name} />
                          {:else}
                            <div class="avatar-placeholder">{request.user.name[0]}</div>
                          {/if}
                        </div>
                        <span><strong>{request.user.name}</strong> (@{request.user.username})</span>
                      </div>
                      
                      <div class="payment-proof">
                        <h4>Payment Proof</h4>
                        <div class="proof-image" on:click={() => viewingImage = request.payment_proof_url}>
                          <img src={request.payment_proof_url} alt="Payment Proof" />
                          <div class="image-overlay">
                            <span>Click to view</span>
                          </div>
                        </div>
                      </div>
                    </div>
                    
                    {#if request.status === 'pending'}
                      <div class="request-actions">
                        <button 
                          class="btn btn-danger" 
                          on:click={() => handlePremiumRequest(request, false)}
                          disabled={loadingStates.premiumRequestAction}
                        >
                          <XCircle size={18} />
                          Reject
                        </button>
                        <button 
                          class="btn btn-success" 
                          on:click={() => handlePremiumRequest(request, true)}
                          disabled={loadingStates.premiumRequestAction}
                        >
                          <CheckCircle size={18} />
                          Approve
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
        
        <!-- Reports Tab -->
        {#if activeTab === 'reports'}
          <div class="tab-panel">
            <div class="panel-header">
              <h2>
                <Flag size={22} />
                <span>Reports</span>
              </h2>
              
              <div class="filter-controls">
                <div class="filter-dropdown">
                  <Filter size={18} />
                  <select bind:value={requestFilter}>
                    <option value="pending">Pending</option>
                    <option value="resolved">Resolved</option>
                    <option value="dismissed">Dismissed</option>
                    <option value="all">All Reports</option>
                  </select>
                </div>
                
                <button class="btn btn-outline btn-icon" on:click={loadReportRequests}>
                  <RefreshCcw size={18} />
                  <span>Refresh</span>
                </button>
              </div>
            </div>
            
            {#if loadingStates.reports}
              <div class="loading-container">
                <div class="loading-spinner"></div>
                <p>Loading reports...</p>
              </div>
            {:else if errors.reports}
              <div class="error-message">
                <AlertTriangle size={20} />
                <p>{errors.reports}</p>
                <button class="btn btn-primary" on:click={loadReportRequests}>Try Again</button>
              </div>
            {:else if reportRequests.length === 0 || !reportRequests.filter(r => requestFilter === 'all' || r.status === requestFilter).length}
              <div class="empty-state">
                <Flag size={40} />
                <p>No reports {requestFilter !== 'all' ? `with status "${requestFilter}"` : ''}</p>
                {#if requestFilter !== 'pending'}
                  <button class="btn btn-outline" on:click={() => requestFilter = 'pending'}>
                    Show Pending Reports
                  </button>
                {/if}
              </div>
            {:else}
              <div class="request-list">
                {#each reportRequests.filter(r => requestFilter === 'all' || r.status === requestFilter) as report (report.id)}
                  <div class="request-card report-card" class:resolved={report.status !== 'pending'}>
                    <div class="request-header">
                      <h3>
                        Report: 
                        {#if report.reported_entity_type === 'thread'}
                          Thread
                        {:else if report.reported_entity_type === 'user'}
                          User
                        {:else if report.reported_entity_type === 'community'}
                          Community
                        {/if}
                      </h3>
                      <div class="request-meta">
                        <span>Reported {formatDate(report.created_at)}</span>
                        
                        {#if report.status !== 'pending'}
                          <span class="badge badge-{report.status === 'resolved' ? 'success' : 'warning'}">
                            {report.status}
                          </span>
                        {/if}
                      </div>
                    </div>
                    
                    <div class="request-body">
                      <div class="report-details">
                        <div class="report-row">
                          <span class="label">Reporter:</span>
                          <span class="value">{report.reporter.name} (@{report.reporter.username})</span>
                        </div>
                        
                        <div class="report-row">
                          <span class="label">Reported {report.reported_entity_type}:</span>
                          <span class="value">{report.reported_entity_name}</span>
                        </div>
                        
                        <div class="report-reason">
                          <span class="label">Reason:</span>
                          <div class="reason-text">{report.reason}</div>
                        </div>
                      </div>
                    </div>
                    
                    {#if report.status === 'pending'}
                      <div class="request-actions">
                        <button 
                          class="btn btn-warning" 
                          on:click={() => handleReportRequest(report, false)}
                          disabled={loadingStates.reportRequestAction}
                        >
                          <XCircle size={18} />
                          Dismiss
                        </button>
                        <button 
                          class="btn btn-success" 
                          on:click={() => handleReportRequest(report, true)}
                          disabled={loadingStates.reportRequestAction}
                        >
                          <CheckCircle size={18} />
                          Resolve
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
        
        <!-- Categories Tab -->
        {#if activeTab === 'categories'}
          <div class="tab-panel">
            <h2>
              <Tag size={22} />
              <span>Manage Categories</span>
            </h2>
            
            {#if errors.action}
              <div class="error-message inline">
                <AlertTriangle size={18} />
                <p>{errors.action}</p>
              </div>
            {/if}
            
            <div class="categories-container">
              <div class="category-section">
                <div class="section-header">
                  <h3>
                    <Hash size={20} />
                    <span>Thread Categories</span>
                  </h3>
                  
                  <button class="btn btn-primary btn-sm" on:click={() => startEditThreadCategory()}>
                    <Plus size={16} />
                    <span>New Thread Category</span>
                  </button>
                </div>
                
                {#if loadingStates.threadCategories}
                  <div class="loading-container slim">
                    <div class="loading-spinner small"></div>
                    <p>Loading thread categories...</p>
                  </div>
                {:else if editingThreadCategory !== null || newThreadCategory.name || newThreadCategory.description}
                  <div class="category-form">
                    <h4>{editingThreadCategory ? 'Edit' : 'New'} Thread Category</h4>
                    <div class="form-group">
                      <label for="thread-cat-name">Name</label>
                      <input 
                        type="text" 
                        id="thread-cat-name" 
                        bind:value={newThreadCategory.name} 
                        placeholder="Category name"
                        maxlength="30"
                      />
                    </div>
                    
                    <div class="form-group">
                      <label for="thread-cat-desc">Description</label>
                      <input 
                        type="text" 
                        id="thread-cat-desc" 
                        bind:value={newThreadCategory.description} 
                        placeholder="Brief description"
                        maxlength="100"
                      />
                    </div>
                    
                    <div class="form-actions">
                      <button 
                        class="btn btn-outline" 
                        on:click={() => startEditThreadCategory(null)} 
                        disabled={loadingStates.categoryAction}
                      >
                        Cancel
                      </button>
                      <button 
                        class="btn btn-primary" 
                        on:click={saveThreadCategory}
                        disabled={loadingStates.categoryAction || !newThreadCategory.name.trim()}
                      >
                        {loadingStates.categoryAction ? 'Saving...' : 'Save Category'}
                      </button>
                    </div>
                  </div>
                {:else if threadCategories.length === 0}
                  <div class="empty-state small">
                    <Tag size={30} />
                    <p>No thread categories found</p>
                    <button class="btn btn-outline" on:click={() => startEditThreadCategory()}>
                      Create First Category
                    </button>
                  </div>
                {:else}
                  <div class="categories-list">
                    <table class="data-table">
                      <thead>
                        <tr>
                          <th>Name</th>
                          <th class="hide-sm">Description</th>
                          <th class="hide-sm">Usage</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {#each threadCategories as category (category.id)}
                          <tr>
                            <td><strong>{category.name}</strong></td>
                            <td class="hide-sm">{category.description}</td>
                            <td class="hide-sm">{category.count} threads</td>
                            <td>
                              <div class="action-buttons compact">
                                <button class="btn btn-icon btn-sm" on:click={() => { newThreadCategory = { name: category.name, description: category.description }; startEditThreadCategory(category); }}>
                                  <Pencil size={16} />
                                  <span class="hide-xs">Edit</span>
                                </button>
                                <button class="btn btn-icon btn-danger btn-sm" on:click={() => deleteThreadCategory(category)}>
                                  <Trash2 size={16} />
                                  <span class="hide-xs">Delete</span>
                                </button>
                              </div>
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  </div>
                {/if}
              </div>
              
              <div class="divider"></div>
              
              <div class="category-section">
                <div class="section-header">
                  <h3>
                    <Building2 size={20} />
                    <span>Community Categories</span>
                  </h3>
                  
                  <button class="btn btn-primary btn-sm" on:click={() => startEditCommunityCategory()}>
                    <Plus size={16} />
                    <span>New Community Category</span>
                  </button>
                </div>
                
                {#if loadingStates.communityCategories}
                  <div class="loading-container slim">
                    <div class="loading-spinner small"></div>
                    <p>Loading community categories...</p>
                  </div>
                {:else if editingCommunityCategory !== null || newCommunityCategory.name || newCommunityCategory.description}
                  <div class="category-form">
                    <h4>{editingCommunityCategory ? 'Edit' : 'New'} Community Category</h4>
                    <div class="form-group">
                      <label for="comm-cat-name">Name</label>
                      <input 
                        type="text" 
                        id="comm-cat-name" 
                        bind:value={newCommunityCategory.name} 
                        placeholder="Category name"
                        maxlength="30"
                      />
                    </div>
                    
                    <div class="form-group">
                      <label for="comm-cat-desc">Description</label>
                      <input 
                        type="text" 
                        id="comm-cat-desc" 
                        bind:value={newCommunityCategory.description} 
                        placeholder="Brief description"
                        maxlength="100"
                      />
                    </div>
                    
                    <div class="form-actions">
                      <button 
                        class="btn btn-outline" 
                        on:click={() => startEditCommunityCategory(null)} 
                        disabled={loadingStates.categoryAction}
                      >
                        Cancel
                      </button>
                      <button 
                        class="btn btn-primary" 
                        on:click={saveCommunityCategory}
                        disabled={loadingStates.categoryAction || !newCommunityCategory.name.trim()}
                      >
                        {loadingStates.categoryAction ? 'Saving...' : 'Save Category'}
                      </button>
                    </div>
                  </div>
                {:else if communityCategories.length === 0}
                  <div class="empty-state small">
                    <Tag size={30} />
                    <p>No community categories found</p>
                    <button class="btn btn-outline" on:click={() => startEditCommunityCategory()}>
                      Create First Category
                    </button>
                  </div>
                {:else}
                  <div class="categories-list">
                    <table class="data-table">
                      <thead>
                        <tr>
                          <th>Name</th>
                          <th class="hide-sm">Description</th>
                          <th class="hide-sm">Usage</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {#each communityCategories as category (category.id)}
                          <tr>
                            <td><strong>{category.name}</strong></td>
                            <td class="hide-sm">{category.description}</td>
                            <td class="hide-sm">{category.count} communities</td>
                            <td>
                              <div class="action-buttons compact">
                                <button class="btn btn-icon btn-sm" on:click={() => { newCommunityCategory = { name: category.name, description: category.description }; startEditCommunityCategory(category); }}>
                                  <Pencil size={16} />
                                  <span class="hide-xs">Edit</span>
                                </button>
                                <button class="btn btn-icon btn-danger btn-sm" on:click={() => deleteCommunityCategory(category)}>
                                  <Trash2 size={16} />
                                  <span class="hide-xs">Delete</span>
                                </button>
                              </div>
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  </div>
                {/if}
              </div>
            </div>
          </div>
        {/if}
        
        <!-- Newsletter Tab -->
        {#if activeTab === 'newsletter'}
          <div class="tab-panel">
            <h2>
              <Mail size={22} />
              <span>Newsletter</span>
            </h2>
            
            {#if !showNewsletter}
              <div class="newsletter-intro">
                <p>Send a newsletter to all users who have opted in to receive updates.</p>
                <div class="newsletter-stats">
                  <div class="stat-card">
                    <BellRing size={24} />
                    <div class="stat-content">
                      <span class="stat-value">{users.filter(u => u.subscribed_to_newsletter).length}</span>
                      <span class="stat-label">Subscribers</span>
                    </div>
                  </div>
                </div>
                
                <button class="btn btn-primary" on:click={toggleNewsletterForm}>
                  <Mail size={18} />
                  <span>Compose Newsletter</span>
                </button>
              </div>
            {:else}
              <div class="newsletter-form">
                <h3>Compose Newsletter</h3>
                
                {#if errors.action}
                  <div class="error-message inline">
                    <AlertTriangle size={18} />
                    <p>{errors.action}</p>
                  </div>
                {/if}
                
                <div class="form-group">
                  <label for="newsletter-subject">Subject</label>
                  <input 
                    type="text" 
                    id="newsletter-subject" 
                    bind:value={newsletterSubject} 
                    placeholder="Newsletter subject"
                    disabled={loadingStates.newsletterAction}
                  />
                </div>
                
                <div class="form-group">
                  <label for="newsletter-body">Content</label>
                  <textarea 
                    id="newsletter-body" 
                    bind:value={newsletterBody} 
                    placeholder="Newsletter content"
                    rows="10"
                    disabled={loadingStates.newsletterAction}
                  ></textarea>
                </div>
                
                <div class="newsletter-actions">
                  <button 
                    class="btn btn-outline" 
                    on:click={toggleNewsletterForm}
                    disabled={loadingStates.newsletterAction}
                  >
                    Cancel
                  </button>
                  <button 
                    class="btn btn-primary" 
                    on:click={sendNewsletter}
                    disabled={loadingStates.newsletterAction || !newsletterSubject.trim() || !newsletterBody.trim()}
                  >
                    <Send size={18} />
                    {loadingStates.newsletterAction ? 'Sending...' : 'Send Newsletter'}
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
  
  {#if viewingImage}
    <div class="image-modal" on:click={() => viewingImage = null}>
      <div class="modal-content" on:click|stopPropagation>
        <button class="close-button" on:click={() => viewingImage = null}>×</button>
        <img src={viewingImage} alt="Full size view" />
      </div>
    </div>
  {/if}

<style lang="scss">
    @use '../styles/variables' as *;

    .admin-page {
    width: 100%;
    max-width: 1200px;
    margin: 0 auto;
    padding: 16px 20px 40px;
    background: var(--background);
    color: var(--text-color);
    
    @media (max-width: 930px) {
        max-width: 100%;
        padding: 16px 16px 40px;
    }
    
    @media (max-width: 480px) {
        padding: 12px 12px 32px;
    }
    }

    .admin-header {
    padding: 16px 0;
    border-bottom: 1px solid var(--border-color);
    margin-bottom: 24px;
    margin-left: -4px;
    
    h1 {
        font-size: 22px;
        font-weight: 800;
        margin: 0;
        color: var(--text-color);
    }
    }

    .admin-container {
    display: grid;
    grid-template-columns: 220px 1fr;
    gap: 24px;
    
    @media (max-width: 768px) {
        grid-template-columns: 1fr;
    }
    }

    .admin-tabs {
    display: flex;
    flex-direction: column;
    gap: 8px;
    position: sticky;
    top: 20px;
    align-self: start;
    background: var(--sidebar-bg);
    padding: 16px;
    border-radius: 8px;
    
    .tab-button {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 12px;
        background: none;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        color: var(--sidebar-text);
        font-size: 14px;
        transition: background 0.2s;
        
        &.active {
        background: var(--section-hover-bg);
        color: var(--sidebar-active-text);
        font-weight: 500;
        }
        
        &:hover:not(.active) {
        background: var(--sidebar-hover-bg);
        }
        
        @media (max-width: 480px) {
        padding: 8px 10px;
        font-size: 13px;
        }
    }
    }

    .tab-content {
    width: 100%;
    }

    .tab-panel {
    padding: 16px;
    background: var(--section-bg);
    border-radius: 8px;
    border: 1px solid var(--border-color);
    
    @media (max-width: 480px) {
        padding: 12px;
    }
    }

    .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    
    h2 {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 18px;
        margin: 0;
        color: var(--text-color);
        
        @media (max-width: 480px) {
        font-size: 16px;
        }
    }
    
    .filter-controls {
        display: flex;
        gap: 8px;
        align-items: center;
        
        @media (max-width: 600px) {
        flex-direction: column;
        gap: 12px;
        align-items: stretch;
        }
    }
    }

    .search-box {
    display: flex;
    align-items: center;
    padding: 6px 10px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background: var(--search-bg);
    color: var(--text-color);
    
    input {
        border: none;
        background: none;
        outline: none;
        width: 150px;
        color: var(--text-color);
        
        @media (max-width: 480px) {
        width: 120px;
        }
    }
    }

    .filter-dropdown {
    display: flex;
    align-items: center;
    padding: 6px 10px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background: var(--search-bg);
    color: var(--text-color);
    
    select {
        border: none;
        background: none;
        outline: none;
        width: 140px;
        color: var(--text-color);
        
        @media (max-width: 480px) {
        width: 110px;
        }
    }
    }

    .btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 12px;
    border-radius: 4px;
    border: none;
    cursor: pointer;
    transition: background 0.2s, color 0.2s, border-color 0.2s;
    
    &.btn-primary {
        background: var(--primary-color);
        color: var(--primary-button-text);
        border: 1px solid var(--primary-color);
        
        &:hover {
        background: var(--primary-color-hover);
        border-color: var(--primary-color-hover);
        }
    }
    
    &.btn-outline {
        background: transparent;
        border: 1px solid var(--border-color);
        color: var(--text-color);
        
        &:hover {
        background: var(--sidebar-hover-bg);
        border-color: var(--sidebar-hover-bg);
        }
    }
    
    &.btn-danger {
        background: var(--error-color);
        color: var(--primary-button-text);
        border: 1px solid var(--error-color);
        
        &:hover {
        background: var(--error-color-hover);
        border-color: var(--error-color-hover);
        }
    }
    
    &.btn-success {
        background: var(--success-color);
        color: var(--primary-button-text);
        border: 1px solid var(--success-color);
        
        &:hover {
        background: var(--success-color-hover);
        border-color: var(--success-color-hover);
        }
    }
    
    &.btn-warning {
        background: #f4b400;
        color: var(--primary-button-text);
        border: 1px solid #f4b400;
        
        &:hover {
        background: darken(#f4b400, 10%);
        border-color: darken(#f4b400, 10%);
        }
    }
    
    &.btn-sm {
        padding: 4px 8px;
        font-size: 12px;
    }
    
    &.btn-icon {
        padding: 4px 8px;
    }
    
    &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }
    }

    .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20px;
    color: var(--text-color);
    
    .loading-spinner {
        width: 40px;
        height: 40px;
        border: 4px solid var(--border-color);
        border-top: 4px solid var(--primary-color);
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    }

    .error-message {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px;
    background: var(--error-bg);
    border-radius: 4px;
    color: var(--error-color);
    
    &.inline {
        margin-bottom: 12px;
    }
    }

    .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20px;
    text-align: center;
    color: var(--secondary-text-color);
    
    &.small {
        padding: 16px;
    }
    }

    .user-list {
    .data-table {
        width: 100%;
        border-collapse: collapse;
        
        th, td {
        padding: 10px;
        text-align: left;
        border-bottom: 1px solid var(--border-color);
        color: var(--text-color);
        }
        
        th {
        background: var(--section-bg);
        font-weight: 600;
        color: var(--text-color);
        }
        
        td.user-cell {
        display: flex;
        align-items: center;
        gap: 10px;
        }
        
        .avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        overflow: hidden;
        display: flex;
        align-items: center;
        justify-content: center;
        
        img {
            width: 100%;
            height: 100%;
            object-fit: cover;
        }
        
        .avatar-placeholder {
            width: 100%;
            height: 100%;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--border-color);
            color: var(--text-color);
            font-weight: 500;
        }
        }
        
        .user-details {
        .user-name {
            font-weight: 500;
            display: block;
            color: var(--text-color);
        }
        
        .user-username {
            color: var(--secondary-text-color);
            font-size: 0.9em;
        }
        }
        
        .status-badges {
        display: flex;
        gap: 6px;
        flex-wrap: wrap;
        
        .badge {
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: 500;
            
            &.badge-success {
            background: var(--success-bg);
            color: var(--success-color);
            }
            
            &.badge-danger {
            background: var(--error-bg);
            color: var(--error-color);
            }
            
            &.badge-premium {
            background: var(--primary-color);
            color: var(--primary-button-text);
            }
            
            &.badge-info {
            background: var(--primary-color);
            color: var(--primary-button-text);
            }
        }
        }
        
        .action-buttons {
        display: flex;
        gap: 6px;
        
        &.compact {
            justify-content: flex-end;
        }
        }
        
        tr.banned {
        opacity: 0.7;
        }
        
        @media (max-width: 768px) {
        .hide-sm {
            display: none;
        }
        }
    }
    }

    .request-list {
    .request-card {
        background: var(--section-bg);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        margin-bottom: 16px;
        padding: 12px;
        color: var(--text-color);
        
        &.resolved {
        opacity: 0.6;
        }
        
        .request-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 12px;
        
        h3 {
            margin: 0;
            font-size: 16px;
            font-weight: 600;
            color: var(--text-color);
        }
        
        .request-meta {
            text-align: right;
            color: var(--secondary-text-color);
            font-size: 0.9em;
            
            .badge {
            margin-top: 4px;
            display: inline-block;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: 500;
            
            &.badge-success {
                background: var(--success-bg);
                color: var(--success-color);
            }
            
            &.badge-danger {
                background: var(--error-bg);
                color: var(--error-color);
            }
            
            &.badge-warning {
                background: #fff3cd;
                color: #856404;
            }
            }
        }
        }
        
        .request-body {
        margin-bottom: 12px;
        
        .request-creator {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 8px;
            color: var(--text-color);
            
            .avatar.small {
            width: 30px;
            height: 30px;
            border-radius: 50%;
            overflow: hidden;
            display: flex;
            align-items: center;
            justify-content: center;
            
            img {
                width: 100%;
                height: 100%;
                object-fit: cover;
            }
            }
        }
        
        .request-description {
            color: var(--text-color);
            margin: 0;
        }
        
        .payment-proof {
            margin-top: 12px;
            
            .proof-image {
            position: relative;
            width: 200px;
            height: 150px;
            overflow: hidden;
            border-radius: 4px;
            cursor: pointer;
            border: 1px solid var(--border-color);
            
            img {
                width: 100%;
                height: 100%;
                object-fit: cover;
            }
            
            .image-overlay {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                background: rgba(0, 0, 0, 0.5);
                display: flex;
                align-items: center;
                justify-content: center;
                color: var(--primary-button-text);
                opacity: 0;
                transition: opacity 0.2s;
            }
            
            &:hover .image-overlay {
                opacity: 1;
            }
            }
        }
        
        .report-details {
            .report-row {
            display: flex;
            gap: 8px;
            margin-bottom: 4px;
            color: var(--text-color);
            
            .label {
                font-weight: 500;
                color: var(--secondary-text-color);
            }
            
            .value {
                flex-grow: 1;
            }
            }
            
            .report-reason {
            margin-top: 8px;
            
            .label {
                font-weight: 500;
                color: var(--secondary-text-color);
            }
            
            .reason-text {
                background: var(--input-bg);
                padding: 8px;
                border-radius: 4px;
                margin-top: 4px;
                color: var(--text-color);
                border: 1px solid var(--border-color);
            }
            }
        }
        }
        
        .request-actions {
        display: flex;
        gap: 8px;
        justify-content: flex-end;
        }
    }
    }

    .categories-container {
    display: grid;
    grid-template-columns: 1fr;
    gap: 24px;
    
    @media (min-width: 768px) {
        grid-template-columns: 1fr 1fr;
    }
    }

    .category-section {
    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        
        h3 {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 16px;
        margin: 0;
        color: var(--text-color);
        }
    }
    
    .category-form {
        background: var(--section-bg);
        padding: 16px;
        border-radius: 8px;
        border: 1px solid var(--border-color);
        color: var(--text-color);
        
        h4 {
        margin: 0 0 12px;
        font-size: 16px;
        color: var(--text-color);
        }
        
        .form-group {
        margin-bottom: 12px;
        
        label {
            display: block;
            margin-bottom: 4px;
            font-weight: 500;
            color: var(--secondary-text-color);
        }
        
        input, textarea {
            width: 100%;
            padding: 6px 10px;
            border: 1px solid var(--border-color);
            border-radius: 4px;
            background: var(--input-bg);
            color: var(--text-color);
            
            &:disabled {
            opacity: 0.6;
            }
        }
        
        textarea {
            resize: vertical;
        }
        }
        
        .form-actions {
        display: flex;
        gap: 8px;
        justify-content: flex-end;
        }
    }
    
    .categories-list {
        .data-table {
        width: 100%;
        border-collapse: collapse;
        
        th, td {
            padding: 10px;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
            color: var(--text-color);
        }
        
        th {
            background: var(--section-bg);
            font-weight: 600;
            color: var(--text-color);
        }
        
        td {
            vertical-align: middle;
        }
        
        .action-buttons {
            display: flex;
            gap: 6px;
            
            .btn-icon {
            padding: 4px;
            background: transparent;
            border: 1px solid var(--border-color);
            color: var(--text-color);
            
            &:hover {
                background: var(--sidebar-hover-bg);
                border-color: var(--sidebar-hover-bg);
            }
            
            &.btn-danger {
                border-color: var(--error-color);
                color: var(--error-color);
                
                &:hover {
                background: var(--error-bg);
                border-color: var(--error-bg);
                }
            }
            }
        }
        
        @media (max-width: 768px) {
            .hide-sm {
            display: none;
            }
        }
        }
    }
    }

    .divider {
    height: 1px;
    background: var(--border-color);
    margin: 24px 0;
    }

    .newsletter-intro {
    padding: 16px;
    background: var(--section-bg);
    border-radius: 8px;
    border: 1px solid var(--border-color);
    color: var(--text-color);
    
    p {
        margin: 0 0 12px;
        color: var(--text-color);
    }
    
    .newsletter-stats {
        display: flex;
        gap: 16px;
        margin-bottom: 16px;
        
        .stat-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        background: var(--input-bg);
        border-radius: 4px;
        flex: 1;
        border: 1px solid var(--border-color);
        
        .stat-content {
            .stat-value {
            font-size: 18px;
            font-weight: 600;
            color: var(--text-color);
            }
            
            .stat-label {
            color: var(--secondary-text-color);
            font-size: 0.9em;
            }
        }
        }
    }
    }

    .newsletter-form {
    background: var(--section-bg);
    padding: 16px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    color: var(--text-color);
    
    h3 {
        margin: 0 0 12px;
        font-size: 16px;
        color: var(--text-color);
    }
    
    .newsletter-actions {
        display: flex;
        gap: 8px;
        justify-content: flex-end;
    }
    }

    .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    
    .modal-content {
        position: relative;
        max-width: 90%;
        max-height: 90%;
        
        img {
        max-width: 100%;
        max-height: 100%;
        object-fit: contain;
        }
        
        .close-button {
        position: absolute;
        top: -10px;
        right: -10px;
        background: var(--error-color);
        color: var(--primary-button-text);
        border: none;
        border-radius: 50%;
        width: 24px;
        height: 24px;
        font-size: 16px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        }
    }
    }
</style>